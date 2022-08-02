// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"strconv"

	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/installer"
	"github.com/elastic/e2e-testing/internal/process"
	log "github.com/sirupsen/logrus"
)

func (fts *FleetTestSuite) processStateChangedOnTheHost(pr string, state string) error {
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
		if pr == common.ElasticAgentProcessName {
			fts.ElasticAgentStopped = true
		}

		return nil
	} else if state != "stopped" {
		return godog.ErrPending
	}

	log.WithFields(log.Fields{
		"service": agentService.Name,
		"process": pr,
	}).Trace("Stopping process on the service")

	err := agentInstaller.Stop(fts.currentContext)
	if err != nil {
		log.WithFields(log.Fields{
			"action":  state,
			"error":   err,
			"service": agentService.Name,
			"process": pr,
		}).Error("Could not stop process on the host")

		return err
	}

	manifest, _ := fts.getDeployer().GetServiceManifest(fts.currentContext, agentService)

	var srv deploy.ServiceRequest
	if fts.StandAlone {
		srv = deploy.NewServiceContainerRequest(manifest.Name)
	} else {
		srv = deploy.NewServiceRequest(manifest.Name)
	}

	return process.CheckState(fts.currentContext, fts.getDeployer(), srv, pr, "stopped", 0)
}

func (fts *FleetTestSuite) processStateOnTheHost(pr string, state string) error {
	ocurrences := "1"
	if state == "uninstalled" || state == "stopped" {
		ocurrences = "0"
	}
	return fts.thereAreInstancesOfTheProcessInTheState(ocurrences, pr, state)
}

func (fts *FleetTestSuite) thereAreInstancesOfTheProcessInTheState(ocurrences string, pr string, state string) error {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.deployer.GetServiceManifest(fts.currentContext, agentService)

	count, err := strconv.Atoi(ocurrences)
	if err != nil {
		return err
	}

	var srv deploy.ServiceRequest
	if fts.StandAlone {
		srv = deploy.NewServiceContainerRequest(manifest.Name)
	} else {
		srv = deploy.NewServiceRequest(manifest.Name)
	}

	return process.CheckState(fts.currentContext, fts.deployer, srv, pr, state, count)
}
