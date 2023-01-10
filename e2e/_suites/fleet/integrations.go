// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

const actionADDED = "added"
const actionREMOVED = "removed"

func (fts *FleetTestSuite) anIntegrationIsSuccessfullyDeployedWithAgentAndInstaller(integration string, installerType string) error {
	err := fts.anAgentIsDeployedToFleetWithInstaller(installerType)
	if err != nil {
		return err
	}

	return fts.theIntegrationIsOperatedInThePolicy(integration, actionADDED)
}

func (fts *FleetTestSuite) theIntegrationIsOperatedInThePolicy(packageName string, action string) error {
	ctx := fts.currentContext
	client := fts.kibanaClient
	policy := fts.Policy

	log.WithFields(log.Fields{
		"action":  action,
		"policy":  policy,
		"package": packageName,
	}).Trace("Doing an operation for a package on a policy")

	integration, err := client.GetIntegrationByPackageName(ctx, packageName)
	if err != nil {
		return err
	}

	if strings.ToLower(action) == actionADDED {
		packageDataStream := kibana.PackageDataStream{
			Name:        fmt.Sprintf("%s-%s", integration.Name, uuid.New().String()),
			Description: integration.Title,
			Namespace:   "default",
			PolicyID:    policy.ID,
			Enabled:     true,
			Package:     integration,
			Inputs:      []kibana.Input{},
		}
		packageDataStream.Inputs = inputs(integration.Name)

		err = client.AddIntegrationToPolicy(ctx, packageDataStream)
		if err != nil {
			log.WithFields(log.Fields{
				"err":       err,
				"packageDS": packageDataStream,
			}).Error("Unable to add integration to policy")
			return err
		}
	} else if strings.ToLower(action) == actionREMOVED {
		packageDataStream, err := client.GetIntegrationFromAgentPolicy(ctx, integration.Name, policy)
		if err != nil {
			return err
		}
		return client.DeleteIntegrationFromPolicy(ctx, packageDataStream)
	}

	return nil
}

func (fts *FleetTestSuite) thePolicyShowsTheDatasourceAdded(packageName string) error {
	log.WithFields(log.Fields{
		"policyID": fts.Policy.ID,
		"package":  packageName,
	}).Trace("Checking if the policy shows the package added")

	maxTimeout := time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	configurationIsPresentFn := func() error {
		packagePolicy, err := fts.kibanaClient.GetIntegrationFromAgentPolicy(fts.currentContext, packageName, fts.Policy)
		if err != nil {
			log.WithFields(log.Fields{
				"packagePolicy": packagePolicy,
				"policy":        fts.Policy,
				"retry":         retryCount,
				"error":         err,
			}).Warn("The integration was not found in the policy")
			retryCount++
			return err
		}

		retryCount++
		return err
	}

	err := backoff.Retry(configurationIsPresentFn, exp)
	if err != nil {
		return err
	}

	return nil
}

func inputs(integration string) []kibana.Input {
	switch integration {
	case "apm":
		return []kibana.Input{
			{
				Type:    "apm",
				Enabled: true,
				Streams: []kibana.Stream{},
				Vars: map[string]kibana.Var{
					"apm-server": {
						Value: "host",
						Type:  "localhost:8200",
					},
				},
			},
		}
	case "linux":
		return []kibana.Input{
			{
				Type:    "linux/metrics",
				Enabled: true,
				Streams: []kibana.Stream{
					{
						ID:      "linux/metrics-linux.memory-" + uuid.New().String(),
						Enabled: true,
						DS: kibana.DataStream{
							Dataset: "linux.memory",
							Type:    "metrics",
						},
						Vars: map[string]kibana.Var{
							"period": {
								Value: "1s",
								Type:  "string",
							},
						},
					},
				},
			},
		}
	case "windows":
		return []kibana.Input{
			{
				Type:    "winlog",
				Enabled: true,
				Streams: []kibana.Stream{
					{
						ID:      "winlog-windows.powershell-" + uuid.New().String(),
						Enabled: true,
						DS: kibana.DataStream{
							Dataset: "windows.powershell",
							Type:    "logs",
						},
						Vars: map[string]kibana.Var{
							"event_id": {
								Value: "some_id",
								Type:  "text",
							},
							"preserve_original_event": {
								Value: "false",
								Type:  "bool",
							},
						},
					},
				},
			},
		}
	}
	return []kibana.Input{}
}
