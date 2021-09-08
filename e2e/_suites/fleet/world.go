// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/elastic/e2e-testing/internal/action"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

// IngestManagerTestSuite represents a test suite, holding references to the pieces needed to run the tests
type IngestManagerTestSuite struct {
	Fleet *FleetTestSuite
}

func (imts *IngestManagerTestSuite) processStateOnTheHost(process string, state string) error {
	ocurrences := "1"
	if state == "uninstalled" || state == "stopped" {
		ocurrences = "0"
	}
	return imts.thereAreInstancesOfTheProcessInTheState(ocurrences, process, state)
}

func (imts *IngestManagerTestSuite) thereAreInstancesOfTheProcessInTheState(ocurrences string, process string, state string) error {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := imts.Fleet.deployer.Inspect(imts.Fleet.currentContext, agentService)

	count, err := strconv.Atoi(ocurrences)
	if err != nil {
		return err
	}

	return CheckProcessState(imts.Fleet.deployer, manifest.Name, process, state, count)
}

// CheckProcessState checks if a process is in the desired state in a container
// name of the container for the service:
// we are using the underlying deployer to run the commands in the container/service
func CheckProcessState(deployer deploy.Deployment, service string, process string, state string, occurrences int) error {
	timeout := time.Duration(utils.TimeoutFactor) * time.Minute
	serviceRequest := deploy.NewServiceRequest(service)

	actionOpts := action.ProcessAction{
		Process:      process,
		DesiredState: state,
		Occurrences:  occurrences,
		MaxTimeout:   timeout}
	waitForProcess, err := action.Attach(imts.Fleet.currentContext, deployer, serviceRequest, action.ActionWaitForProcess, actionOpts)
	if err != nil {
		log.WithField("error", err).Error("Unable to attach Process check action")
	}

	_, err = waitForProcess.Run(imts.Fleet.currentContext)
	if err != nil {
		if state == "started" {
			log.WithFields(log.Fields{
				"service ": service,
				"error":    err,
				"timeout":  timeout,
			}).Error("The process was not found but should be present")
		} else {
			log.WithFields(log.Fields{
				"service": service,
				"error":   err,
				"timeout": timeout,
			}).Error("The process was found but shouldn't be present")
		}

		return err
	}

	return nil
}
