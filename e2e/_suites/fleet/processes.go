// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/internal/action"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/installer"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

// CheckProcessState checks if a process is in the desired state in a container
// name of the container for the service:
// we are using the underlying deployer to run the commands in the container/service
func CheckProcessState(ctx context.Context, deployer deploy.Deployment, service deploy.ServiceRequest, process string, state string, occurrences int) error {
	timeout := time.Duration(utils.TimeoutFactor) * time.Minute

	if runtime.GOOS == "windows" {
		process = fmt.Sprintf("%s.exe", process)
	}

	actionOpts := action.ProcessAction{
		Process:      process,
		DesiredState: state,
		Occurrences:  occurrences,
		MaxTimeout:   timeout}
	waitForProcess, err := action.Attach(ctx, deployer, service, action.ActionWaitForProcess, actionOpts)
	if err != nil {
		log.WithField("error", err).Error("Unable to attach Process check action")
	}

	_, err = waitForProcess.Run(ctx)
	if err != nil {
		if state == "started" {
			log.WithFields(log.Fields{
				"service ": service,
				"error":    err,
				"process ": process,
				"timeout":  timeout,
			}).Error("The process was not found but should be present")
		} else {
			log.WithFields(log.Fields{
				"service":  service,
				"error":    err,
				"process ": process,
				"timeout":  timeout,
			}).Error("The process was found but shouldn't be present")
		}

		return err
	}

	return nil
}

func (fts *FleetTestSuite) processStateChangedOnTheHost(process string, state string) error {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	agentInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, fts.InstallerType)
	if state == "started" {
		err := agentInstaller.Start(fts.currentContext)
		return err
	} else if state == "restarted" {
		err := agentInstaller.Restart(fts.currentContext)
		if err != nil {
			return err
		}

		return nil
	} else if state == "uninstalled" {
		err := agentInstaller.Uninstall(fts.currentContext)
		if err != nil {
			return err
		}

		// signal that the elastic-agent was uninstalled
		if process == common.ElasticAgentProcessName {
			fts.ElasticAgentStopped = true
		}

		return nil
	} else if state != "stopped" {
		return godog.ErrPending
	}

	log.WithFields(log.Fields{
		"service": agentService.Name,
		"process": process,
	}).Trace("Stopping process on the service")

	err := agentInstaller.Stop(fts.currentContext)
	if err != nil {
		log.WithFields(log.Fields{
			"action":  state,
			"error":   err,
			"service": agentService.Name,
			"process": process,
		}).Error("Could not stop process on the host")

		return err
	}

	manifest, _ := fts.getDeployer().Inspect(fts.currentContext, agentService)

	var srv deploy.ServiceRequest
	if fts.StandAlone {
		srv = deploy.NewServiceContainerRequest(manifest.Name)
	} else {
		srv = deploy.NewServiceRequest(manifest.Name)
	}

	return CheckProcessState(fts.currentContext, fts.getDeployer(), srv, process, "stopped", 0)
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

	var srv deploy.ServiceRequest
	if imts.Fleet.StandAlone {
		srv = deploy.NewServiceContainerRequest(manifest.Name)
	} else {
		srv = deploy.NewServiceRequest(manifest.Name)
	}

	return CheckProcessState(imts.Fleet.currentContext, imts.Fleet.deployer, srv, process, state, count)
}
