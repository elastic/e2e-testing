// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/compose"
	"github.com/elastic/e2e-testing/internal/docker"
	"github.com/elastic/e2e-testing/internal/elasticsearch"
	"github.com/elastic/e2e-testing/internal/installer"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// StandAloneTestSuite represents the scenarios for Stand-alone-mode
type StandAloneTestSuite struct {
	Cleanup  bool
	Hostname string
	Image    string
	// date controls for queries
	AgentStoppedDate             time.Time
	RuntimeDependenciesStartDate time.Time
	kibanaClient                 *kibana.Client
}

// afterScenario destroys the state created by a scenario
func (sats *StandAloneTestSuite) afterScenario() {
	serviceManager := compose.NewServiceManager()
	serviceName := common.ElasticAgentServiceName

	if log.IsLevelEnabled(log.DebugLevel) {
		_ = sats.getContainerLogs()
	}

	developerMode := shell.GetEnvBool("DEVELOPER_MODE")
	if !developerMode {
		_ = serviceManager.RemoveServicesFromCompose(context.Background(), common.FleetProfileName, []string{serviceName}, common.ProfileEnv)
	} else {
		log.WithField("service", serviceName).Info("Because we are running in development mode, the service won't be stopped")
	}
}

func (sats *StandAloneTestSuite) contributeSteps(s *godog.ScenarioContext) {
	s.Step(`^a "([^"]*)" stand-alone agent is deployed$`, sats.aStandaloneAgentIsDeployed)
	s.Step(`^a "([^"]*)" stand-alone agent is deployed with fleet server mode$`, sats.aStandaloneAgentIsDeployedWithFleetServerMode)
	s.Step(`^there is new data in the index from agent$`, sats.thereIsNewDataInTheIndexFromAgent)
	s.Step(`^the "([^"]*)" docker container is stopped$`, sats.theDockerContainerIsStopped)
	s.Step(`^there is no new data in the index after agent shuts down$`, sats.thereIsNoNewDataInTheIndexAfterAgentShutsDown)
	s.Step(`^the stand-alone agent is listed in Fleet as "([^"]*)"$`, sats.theStandaloneAgentIsListedInFleetWithStatus)
}

func (sats *StandAloneTestSuite) theStandaloneAgentIsListedInFleetWithStatus(desiredStatus string) error {
	waitForAgents := func() error {
		agents, err := sats.kibanaClient.ListAgents()
		if err != nil {
			return err
		}

		if len(agents) == 0 {
			return errors.New("No agents found")
		}

		agentZero := agents[0]
		hostname := agentZero.LocalMetadata.Host.HostName

		return theAgentIsListedInFleetWithStatus(desiredStatus, hostname)
	}
	maxTimeout := time.Duration(common.TimeoutFactor) * time.Minute * 2
	exp := common.GetExponentialBackOff(maxTimeout)

	err := backoff.Retry(waitForAgents, exp)
	if err != nil {
		return err
	}
	return nil
}

func (sats *StandAloneTestSuite) aStandaloneAgentIsDeployedWithFleetServerMode(image string) error {
	return sats.startAgent(image, map[string]string{"fleetServerMode": "1"})
}

func (sats *StandAloneTestSuite) aStandaloneAgentIsDeployed(image string) error {
	return sats.startAgent(image, nil)
}

func (sats *StandAloneTestSuite) startAgent(image string, env map[string]string) error {

	log.Trace("Deploying an agent to Fleet")

	dockerImageTag := common.AgentVersion

	useCISnapshots := shell.GetEnvBool("BEATS_USE_CI_SNAPSHOTS")
	beatsLocalPath := shell.GetEnv("BEATS_LOCAL_PATH", "")
	if useCISnapshots || beatsLocalPath != "" {
		// load the docker images that were already:
		// a. downloaded from the GCP bucket
		// b. fetched from the local beats binaries
		dockerInstaller := installer.GetElasticAgentInstaller("docker", image, common.AgentVersion)

		dockerInstaller.PreInstallFn()

		dockerImageTag += "-amd64"
	}

	serviceManager := compose.NewServiceManager()

	common.ProfileEnv["elasticAgentDockerImageSuffix"] = ""
	if image != "default" {
		common.ProfileEnv["elasticAgentDockerImageSuffix"] = "-" + image
	}

	common.ProfileEnv["elasticAgentDockerNamespace"] = utils.GetDockerNamespaceEnvVar()

	containerName := fmt.Sprintf("%s_%s_%d", common.FleetProfileName, common.ElasticAgentServiceName, 1)

	common.ProfileEnv["elasticAgentContainerName"] = containerName
	common.ProfileEnv["elasticAgentPlatform"] = "linux/amd64"
	common.ProfileEnv["elasticAgentTag"] = dockerImageTag

	for k, v := range env {
		common.ProfileEnv[k] = v
	}

	err := serviceManager.AddServicesToCompose(context.Background(), common.FleetProfileName, []string{common.ElasticAgentServiceName}, common.ProfileEnv)
	if err != nil {
		log.Error("Could not deploy the elastic-agent")
		return err
	}

	// get container hostname once
	hostname, err := docker.GetContainerHostname(containerName)
	if err != nil {
		return err
	}

	sats.Image = image
	sats.Hostname = hostname
	sats.Cleanup = true

	err = sats.installTestTools(containerName)
	if err != nil {
		return err
	}

	return nil
}

func (sats *StandAloneTestSuite) getContainerLogs() error {
	serviceManager := compose.NewServiceManager()

	profile := common.FleetProfileName
	serviceName := common.ElasticAgentServiceName

	composes := []string{
		profile,     // profile name
		serviceName, // agent service
	}
	err := serviceManager.RunCommand(profile, composes, []string{"logs", serviceName}, common.ProfileEnv)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"service": serviceName,
		}).Error("Could not retrieve Elastic Agent logs")

		return err
	}

	return nil
}

// installTestTools we need the container name because we use the Docker Client instead of Docker Compose
// we are going to install those tools we use in the test framework for checking
// and verifications
func (sats *StandAloneTestSuite) installTestTools(containerName string) error {
	if sats.Image != "ubi8" {
		return nil
	}

	cmd := []string{"microdnf", "install", "procps-ng"}

	log.WithFields(log.Fields{
		"command":       cmd,
		"containerName": containerName,
	}).Trace("Installing test tools ")

	_, err := docker.ExecCommandIntoContainer(context.Background(), containerName, "root", cmd)
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

func (sats *StandAloneTestSuite) thereIsNewDataInTheIndexFromAgent() error {
	maxTimeout := time.Duration(common.TimeoutFactor) * time.Minute * 2
	minimumHitsCount := 50

	result, err := searchAgentData(sats.Hostname, sats.RuntimeDependenciesStartDate, minimumHitsCount, maxTimeout)
	if err != nil {
		return err
	}

	log.Tracef("Search result: %v", result)

	return elasticsearch.AssertHitsArePresent(result)
}

func (sats *StandAloneTestSuite) theDockerContainerIsStopped(serviceName string) error {
	serviceManager := compose.NewServiceManager()

	err := serviceManager.RemoveServicesFromCompose(context.Background(), common.FleetProfileName, []string{serviceName}, common.ProfileEnv)
	if err != nil {
		return err
	}
	sats.AgentStoppedDate = time.Now().UTC()

	return nil
}

func (sats *StandAloneTestSuite) thereIsNoNewDataInTheIndexAfterAgentShutsDown() error {
	maxTimeout := time.Duration(30) * time.Second
	minimumHitsCount := 1

	result, err := searchAgentData(sats.Hostname, sats.AgentStoppedDate, minimumHitsCount, maxTimeout)
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

func searchAgentData(hostname string, startDate time.Time, minimumHitsCount int, maxTimeout time.Duration) (elasticsearch.SearchResult, error) {
	timezone := "America/New_York"

	esQuery := map[string]interface{}{
		"version": true,
		"size":    500,
		"docvalue_fields": []map[string]interface{}{
			{
				"field":  "@timestamp",
				"format": "date_time",
			},
			{
				"field":  "system.process.cpu.start_time",
				"format": "date_time",
			},
			{
				"field":  "system.service.state_since",
				"format": "date_time",
			},
		},
		"_source": map[string]interface{}{
			"excludes": []map[string]interface{}{},
		},
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{},
				"filter": []map[string]interface{}{
					{
						"bool": map[string]interface{}{
							"filter": []map[string]interface{}{
								{
									"bool": map[string]interface{}{
										"should": []map[string]interface{}{
											{
												"match_phrase": map[string]interface{}{
													"host.name": hostname,
												},
											},
										},
										"minimum_should_match": 1,
									},
								},
								{
									"bool": map[string]interface{}{
										"should": []map[string]interface{}{
											{
												"range": map[string]interface{}{
													"@timestamp": map[string]interface{}{
														"gte":       startDate,
														"time_zone": timezone,
													},
												},
											},
										},
										"minimum_should_match": 1,
									},
								},
							},
						},
					},
					{
						"range": map[string]interface{}{
							"@timestamp": map[string]interface{}{
								"gte":    startDate,
								"format": "strict_date_optional_time",
							},
						},
					},
				},
				"should":   []map[string]interface{}{},
				"must_not": []map[string]interface{}{},
			},
		},
	}

	indexName := "logs-elastic_agent-default"

	result, err := elasticsearch.WaitForNumberOfHits(context.Background(), indexName, esQuery, minimumHitsCount, maxTimeout)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn(elasticsearch.WaitForIndices())
	}

	return result, err
}
