// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

func (fts *FleetTestSuite) getAgentDefaultAPIKey() (string, error) {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.getDeployer().GetServiceManifest(fts.currentContext, agentService)
	agent, err := fts.kibanaClient.GetAgentByHostnameFromList(fts.currentContext, manifest.Hostname)
	if err != nil {
		return "", err
	}
	return agent.DefaultAPIKey, nil
}

func (fts *FleetTestSuite) theAgentGetDefaultAPIKey() error {
	defaultAPIKey, _ := fts.getAgentDefaultAPIKey()
	log.WithFields(log.Fields{
		"default_api_key": defaultAPIKey,
	}).Info("The Agent is installed with Default Api Key")
	fts.DefaultAPIKey = defaultAPIKey
	return nil
}

func (fts *FleetTestSuite) verifyDefaultAPIKey(status string) error {
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	checkAPIKeyFn := func() error {
		newDefaultAPIKey, _ := fts.getAgentDefaultAPIKey()

		logFields := log.Fields{
			"new_default_api_key": newDefaultAPIKey,
			"old_default_api_key": fts.DefaultAPIKey,
		}

		defaultAPIKeyHasChanged := (newDefaultAPIKey != fts.DefaultAPIKey)

		if status == "changed" {
			if !defaultAPIKeyHasChanged {
				retryCount++
				log.WithFields(logFields).Warn("Integration added and Default API Key did not change yet")
				return fmt.Errorf("integration added and Default API Key did not change yet")
			}

			log.WithFields(logFields).Infof("Default API Key has %s when the Integration has been added", status)
			return nil
		}

		if status == "not changed" {
			if defaultAPIKeyHasChanged {
				retryCount++
				log.WithFields(logFields).Error("Integration updated and Default API Key is still changed")
				return fmt.Errorf("integration updated and Default API Key is still changed")
			}

			log.WithFields(logFields).Infof("Default API Key has %s when the Integration has been updated", status)
			return nil
		}

		log.Warnf("Status %s is not supported yet", status)
		return godog.ErrPending
	}

	err := backoff.Retry(checkAPIKeyFn, exp)
	return err
}
