// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/installer"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/elastic/e2e-testing/pkg/downloads"
	"github.com/pkg/errors"

	"github.com/elastic/e2e-testing/internal/elasticsearch"
	log "github.com/sirupsen/logrus"
)

func (fts *FleetTestSuite) aStandaloneAgentIsDeployed(image string) error {
	return fts.startStandAloneAgent(image, false)
}

func (fts *FleetTestSuite) bootstrapFleetServerFromAStandaloneAgent(image string) error {
	return fts.startStandAloneAgent(image, true)
}

func (fts *FleetTestSuite) theDockerContainerIsStopped(serviceName string) error {
	agentService := deploy.NewServiceContainerRequest(serviceName)
	err := fts.getDeployer().Stop(fts.currentContext, agentService)
	if err != nil {
		return err
	}
	fts.AgentStoppedDate = time.Now().UTC()

	return nil
}

func (fts *FleetTestSuite) theStandaloneAgentIsListedInFleetWithStatus(desiredStatus string) error {
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	exp := utils.GetExponentialBackOff(maxTimeout)
	retryCount := 0

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.getDeployer().GetServiceManifest(fts.currentContext, agentService)

	waitForAgents := func() error {
		retryCount++

		agents, err := fts.kibanaClient.ListAgents(fts.currentContext)
		if err != nil {
			return err
		}

		if len(agents) == 0 {
			return errors.New("No agents found")
		}

		for _, agent := range agents {
			hostname := agent.LocalMetadata.Host.HostName

			if hostname == manifest.Hostname {
				return theAgentIsListedInFleetWithStatus(fts.currentContext, desiredStatus, hostname)
			}
		}

		err = errors.New("Agent not found in Fleet")
		log.WithFields(log.Fields{
			"elapsedTime": exp.GetElapsedTime(),
			"hostname":    manifest.Hostname,
			"retries":     retryCount,
		}).Warn(err)

		return err
	}

	err := backoff.Retry(waitForAgents, exp)
	if err != nil {
		return err
	}
	return nil
}

func (fts *FleetTestSuite) thereIsNoNewDataInTheIndexAfterAgentShutsDown() error {
	maxTimeout := time.Duration(30) * time.Second
	minimumHitsCount := 1

	agentService := deploy.NewServiceContainerRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.getDeployer().GetServiceManifest(fts.currentContext, agentService)
	result, err := searchAgentData(fts.currentContext, manifest.Hostname, fts.AgentStoppedDate, minimumHitsCount, maxTimeout)
	if err != nil {
		if strings.Contains(err.Error(), "type:index_not_found_exception") {
			return err
		}

		log.WithFields(log.Fields{
			"error": err,
		}).Info("No documents were found for the Agent in the index after it stopped")
		return nil
	}

	return elasticsearch.AssertHitsAreNotPresent(result)
}

func (fts *FleetTestSuite) startStandAloneAgent(image string, bootstrapFleetServer bool) error {
	fts.StandAlone = true
	log.Trace("Deploying an agent to Fleet")

	dockerImageTag := common.ElasticAgentVersion

	common.ProfileEnv["elasticAgentDockerNamespace"] = deploy.GetDockerNamespaceEnvVar("beats")
	common.ProfileEnv["elasticAgentDockerImageSuffix"] = ""
	if image != "default" {
		common.ProfileEnv["elasticAgentDockerImageSuffix"] = "-" + image
	}

	if downloads.UseElasticAgentCISnapshots() {
		// load the docker images that were already:
		// a. downloaded from the GCP bucket
		// b. fetched from the local beats binaries
		agentService := deploy.NewServiceContainerRequest(common.ElasticAgentServiceName)
		dockerInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, "docker")
		dockerInstaller.Preinstall(fts.currentContext)

		arch := utils.GetArchitecture()
		dockerImageTag += "-" + arch
	}

	// Grab a new enrollment key for new agent
	enrollmentKey, err := fts.kibanaClient.CreateEnrollmentAPIKey(fts.currentContext, fts.Policy)
	if err != nil {
		return err
	}
	fts.CurrentToken = enrollmentKey.APIKey
	fts.CurrentTokenID = enrollmentKey.ID

	cfg, err := kibana.NewFleetConfig(fts.CurrentToken)
	if err != nil {
		return err
	}

	// See https://github.com/elastic/beats/blob/4accfa8/x-pack/elastic-agent/pkg/agent/cmd/container.go#L73-L85
	// to understand the environment variables used by the elastic-agent to automatically
	// enroll the new agent container in Fleet
	common.ProfileEnv["fleetInsecure"] = "1"
	common.ProfileEnv["fleetUrl"] = cfg.FleetServerURL()
	common.ProfileEnv["fleetEnroll"] = "1"
	common.ProfileEnv["fleetEnrollmentToken"] = cfg.EnrollmentToken

	common.ProfileEnv["fleetServerPort"] = "8221" // fixed port to avoid collitions with the stack's fleet-server

	common.ProfileEnv["elasticAgentTag"] = dockerImageTag

	if bootstrapFleetServer {
		common.ProfileEnv["fleetServerMode"] = "1"
	} else {
		common.ProfileEnv["fleetServerMode"] = "0"
	}

	agentService := deploy.NewServiceContainerRequest(common.ElasticAgentServiceName)

	err = fts.getDeployer().Add(fts.currentContext, deploy.NewServiceContainerRequest(common.FleetProfileName), []deploy.ServiceRequest{agentService}, common.ProfileEnv)
	if err != nil {
		log.Error("Could not deploy the elastic-agent")
		return err
	}

	fts.Image = image

	manifest, _ := fts.getDeployer().GetServiceManifest(fts.currentContext, agentService)

	err = fts.installTestTools(manifest.Name)
	if err != nil {
		return err
	}

	return nil
}

// installTestTools we need the container name because we use the Docker Client instead of Docker Compose
// we are going to install those tools we use in the test framework for checking
// and verifications
func (fts *FleetTestSuite) installTestTools(containerName string) error {
	if fts.Image != "ubi8" {
		return nil
	}

	cmd := []string{"microdnf", "install", "procps-ng"}

	log.WithFields(log.Fields{
		"command":       cmd,
		"containerName": containerName,
	}).Trace("Installing test tools ")

	_, err := deploy.ExecCommandIntoContainer(fts.currentContext, containerName, "root", cmd)
	if err != nil {
		log.WithFields(log.Fields{
			"command":       cmd,
			"containerName": containerName,
			"error":         err,
		}).Error("Could not install test tools using the Docker client")
		return err
	}

	log.WithFields(log.Fields{
		"command":       cmd,
		"containerName": containerName,
	}).Debug("Test tools installed")

	return nil
}
