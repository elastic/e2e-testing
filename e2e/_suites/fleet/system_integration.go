// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (fts *FleetTestSuite) getAgentOSData() (string, error) {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.getDeployer().Inspect(fts.currentContext, agentService)
	agent, err := fts.kibanaClient.GetAgentByHostnameFromList(fts.currentContext, manifest.Hostname)
	if err != nil {
		return "", err
	}
	return agent.LocalMetadata.OS.Platform, nil
}

func (fts *FleetTestSuite) theMetricsInTheDataStream(name string, set string) error {
	timeNow := time.Now()
	startTime := timeNow.Unix()

	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	os, _ := fts.getAgentOSData()

	waitForDataStreams := func() error {
		dataStreams, _ := fts.kibanaClient.GetDataStreams(fts.currentContext)

		for _, item := range dataStreams.Children() {
			if item.Path("dataset").Data().(string) == "system."+set {
				log.WithFields(log.Fields{
					"dataset":     "system." + set,
					"elapsedTime": exp.GetElapsedTime(),
					"enabled":     "true",
					"retries":     retryCount,
					"type":        name,
					"os":          os,
				}).Info("The " + name + " with value system." + set + " in the metrics")

				if int64(int64(item.Path("last_activity_ms").Data().(float64))) > startTime {
					log.WithFields(log.Fields{
						"elapsedTime":      exp.GetElapsedTime(),
						"last_activity_ms": item.Path("last_activity_ms").Data().(float64),
						"retries":          retryCount,
						"startTime":        startTime,
						"os":               os,
					}).Info("The " + name + " with value system." + set + " in the metrics")
				}

				return nil
			}
		}

		err := errors.New("No " + name + " with value system." + set + " found in the metrics")

		log.WithFields(log.Fields{
			"elapsedTime": exp.GetElapsedTime(),
			"name":        name,
			"retry":       retryCount,
			"set":         set,
		}).Warn(err.Error())

		retryCount++

		return err
	}

	err := backoff.Retry(waitForDataStreams, exp)
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) thePolicyIsUpdatedToHaveSystemSet(name string, set string) error {
	if name != "linux/metrics" && name != "system/metrics" && name != "logfile" && name != "log" {
		log.WithFields(log.Fields{
			"name": name,
		}).Warn("We only support system system/metrics, log, logfile and linux/metrics policy to be updated")
		return godog.ErrPending
	}

	var err error
	var packageDS kibana.PackageDataStream
	var kibanaInputs []kibana.Input
	var metrics = ""

	if name == "linux/metrics" {
		metrics = "linux"
		packageDS, err = fts.kibanaClient.GetIntegrationFromAgentPolicy(fts.currentContext, metrics, fts.Policy)
		if err != nil {
			return err
		}

		kibanaInputs = metricsInputs(name, set, "/linux_metrics.json", metrics)
	} else if name == "system/metrics" || name == "logfile" || name == "log" {
		metrics = "system"
		packageDS, err = fts.kibanaClient.GetIntegrationFromAgentPolicy(fts.currentContext, metrics, fts.Policy)
		if err != nil {
			return err
		}

		packagePolicy, errPolicy := fts.kibanaClient.GetPackagePolicy(fts.currentContext, packageDS.ID)
		if errPolicy != nil {
			return errPolicy
		}

		kibanaInputs = packagePolicy.Inputs
		log.WithFields(log.Fields{
			"inputs": packagePolicy.Inputs,
		}).Trace("Inputs from the package policy")
	} else {
		log.WithFields(log.Fields{
			"type":    name,
			"dataset": set,
		}).Warn("Package Policy not supported yet")
		return godog.ErrPending
	}

	os, _ := fts.getAgentOSData()

	fts.Integration = packageDS.Package

	log.WithFields(log.Fields{
		"type":    name,
		"dataset": metrics + "." + set,
	}).Info("Getting information about Policy package type " + name + " name with dataset " + metrics + "." + set)

	for _, item := range packageDS.Inputs {
		if item.Type == name {
			packageDS.Inputs = kibanaInputs
		}
	}
	log.WithFields(log.Fields{
		"inputs": packageDS.Inputs,
	}).Info("Updating integration package config")

	updatedAt, err := fts.kibanaClient.UpdateIntegrationPackagePolicy(fts.currentContext, packageDS)
	if err != nil {
		return err
	}

	fts.PolicyUpdatedAt = updatedAt

	log.WithFields(log.Fields{
		"dataset": metrics + "." + set,
		"enabled": "true",
		"type":    "metrics",
		"os":      os,
	}).Info("Policy Updated with package name " + metrics + "." + set)

	return nil
}
