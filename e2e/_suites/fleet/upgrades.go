// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/installer"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/elastic/e2e-testing/pkg/downloads"
	log "github.com/sirupsen/logrus"
)

const (
	upgradeMaxTimeout = 10 * time.Minute
)

func (fts *FleetTestSuite) agentInVersion(version string) error {
	switch version {
	case "latest":
		version = downloads.GetSnapshotVersion(common.ElasticAgentVersion)
	}
	log.Tracef("Checking if agent is in version %s. Current version: %s", version, fts.Version)

	retryCount := 0
	maxTimeout := upgradeMaxTimeout
	exp := utils.GetExponentialBackOff(maxTimeout)

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.getDeployer().GetServiceManifest(fts.currentContext, agentService)

	agentInVersionFn := func() error {
		retryCount++

		agent, err := fts.kibanaClient.GetAgentByHostname(fts.currentContext, manifest.Hostname)
		if err != nil {
			log.WithFields(log.Fields{
				"agent":       agent,
				"error":       err,
				"maxTimeout":  maxTimeout,
				"elapsedTime": exp.GetElapsedTime(),
				"retries":     retryCount,
				"version":     version,
			}).Warn("Could not get agent by hostname")
			return err
		}

		retrievedVersion := agent.LocalMetadata.Elastic.Agent.Version
		if isSnapshot := agent.LocalMetadata.Elastic.Agent.Snapshot; isSnapshot {
			retrievedVersion += "-SNAPSHOT"
		}

		if retrievedVersion != version {
			err := fmt.Errorf("version mismatch required '%s' retrieved '%s'", version, retrievedVersion)
			log.WithFields(log.Fields{
				"elapsedTime":      exp.GetElapsedTime(),
				"error":            err,
				"maxTimeout":       maxTimeout,
				"retries":          retryCount,
				"retrievedVersion": retrievedVersion,
				"version":          version,
			}).Warn("Version mismatch")
			return err
		}

		return nil
	}

	return backoff.Retry(agentInVersionFn, exp)
}

func (fts *FleetTestSuite) anAgentIsUpgradedToVersion(desiredVersion string) error {
	switch desiredVersion {
	case "latest":
		desiredVersion = common.ElasticAgentVersion
	}
	log.Tracef("Desired version is %s. Current version: %s", desiredVersion, fts.Version)

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)

	/*
		// upgrading using the command is needed for stand-alone mode, only
		agentInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, fts.InstallerType)

		log.Tracef("Upgrading agent from %s to %s with 'upgrade' command.", desiredVersion, fts.Version)
		return agentInstaller.Upgrade(fts.currentContext, desiredVersion)
	*/

	manifest, _ := fts.getDeployer().GetServiceManifest(fts.currentContext, agentService)
	return fts.kibanaClient.UpgradeAgent(fts.currentContext, manifest.Hostname, desiredVersion)
}

func (fts *FleetTestSuite) anStaleAgentIsDeployedToFleetWithInstaller(staleVersion string, installerType string) error {
	switch staleVersion {
	case "latest":
		staleVersion = common.ElasticAgentVersion
	}

	fts.Version = staleVersion

	log.Tracef("The stale version is %s", fts.Version)

	return fts.anAgentIsDeployedToFleetWithInstaller(installerType)
}

func (fts *FleetTestSuite) installCerts() error {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	agentInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, fts.InstallerType)

	err := agentInstaller.InstallCerts(fts.currentContext)
	if err != nil {
		log.WithFields(log.Fields{
			"agentVersion":      common.ElasticAgentVersion,
			"agentStaleVersion": fts.Version,
			"error":             err,
			"installer":         agentInstaller,
		}).Error("Could not install the certificates")
		return err
	}

	log.WithFields(log.Fields{
		"agentVersion":      common.ElasticAgentVersion,
		"agentStaleVersion": fts.Version,
		"error":             err,
		"installer":         agentInstaller,
	}).Tracef("Certs were installed")
	return nil
}
