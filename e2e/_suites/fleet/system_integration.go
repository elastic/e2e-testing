// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/cenkalti/backoff/v4"
	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (fts *FleetTestSuite) theMetricsInTheDataStream(name string, set string) error {
	timeNow := time.Now()
	startTime := timeNow.Unix()

	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

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
				}).Info("The " + name + " with value system." + set + " in the metrics")

				if int64(int64(item.Path("last_activity_ms").Data().(float64))) > startTime {
					log.WithFields(log.Fields{
						"elapsedTime":      exp.GetElapsedTime(),
						"last_activity_ms": item.Path("last_activity_ms").Data().(float64),
						"retries":          retryCount,
						"startTime":        startTime,
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
	} else if name == "winlog" {
		metrics = "windows"
		packageDS, err = fts.kibanaClient.GetIntegrationFromAgentPolicy(fts.currentContext, metrics, fts.Policy)
		if err != nil {
			return err
		}

		kibanaInputs = metricsInputs(name, set, "/windows.json", metrics)
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
	}).Info("Policy Updated with package name " + metrics + "." + set)

	return nil
}

func metricsInputs(integration string, set string, file string, metrics string) []kibana.Input {
	metricsFile := filepath.Join(testResourcesDir, file)
	jsonData, err := readJSONFile(metricsFile)
	if err != nil {
		log.Warnf("An error happened while reading metrics file, returning an empty array of inputs: %v", err)
		return []kibana.Input{}
	}

	data := parseJSONMetrics(jsonData, integration, set, metrics)
	return []kibana.Input{
		{
			Type:    integration,
			Enabled: true,
			Streams: data,
		},
	}

	return []kibana.Input{}
}

func parseJSONMetrics(data *gabs.Container, integration string, set string, metrics string) []kibana.Stream {
	for i, item := range data.Children() {
		if item.Path("type").Data().(string) == integration {
			for idx, stream := range item.S("streams").Children() {
				dataSet, _ := stream.Path("data_stream.dataset").Data().(string)
				if dataSet == metrics+"."+set {
					data.SetP(
						integration+"-"+metrics+"."+set+"-"+uuid.New().String(),
						fmt.Sprintf("inputs.%d.streams.%d.id", i, idx),
					)
					data.SetP(
						true,
						fmt.Sprintf("inputs.%d.streams.%d.enabled", i, idx),
					)

					var dataStreamOut []kibana.Stream
					if err := json.Unmarshal(data.Path(fmt.Sprintf("inputs.%d.streams", i)).Bytes(), &dataStreamOut); err != nil {
						return []kibana.Stream{}
					}

					return dataStreamOut
				}
			}
		}
	}
	return []kibana.Stream{}
}

func readJSONFile(file string) (*gabs.Container, error) {
	jsonFile, err := os.Open(file)
	if err != nil {
		fmt.Println(err)
	}
	log.WithFields(log.Fields{
		"file": file,
	}).Info("Successfully Opened " + file)

	defer jsonFile.Close()
	data, err := os.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	jsonParsed, err := gabs.ParseJSON(data)
	if err != nil {
		return nil, err
	}

	return jsonParsed.S("inputs"), nil
}
