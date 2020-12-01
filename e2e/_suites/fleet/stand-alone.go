// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/cli/docker"
	"github.com/elastic/e2e-testing/cli/services"
	"github.com/elastic/e2e-testing/e2e"
	log "github.com/sirupsen/logrus"
)

// StandAloneTestSuite represents the scenarios for Stand-alone-mode
type StandAloneTestSuite struct {
	AgentConfigFilePath string
	Cleanup             bool
	Hostname            string
	Image               string
	// date controls for queries
	AgentStoppedDate             time.Time
	RuntimeDependenciesStartDate time.Time
}

// afterScenario destroys the state created by a scenario
func (sats *StandAloneTestSuite) afterScenario() {
	serviceManager := services.NewServiceManager()
	serviceName := ElasticAgentServiceName

	if log.IsLevelEnabled(log.DebugLevel) {
		_ = sats.getContainerLogs()
	}

	if !developerMode {
		_ = serviceManager.RemoveServicesFromCompose(FleetProfileName, []string{serviceName}, profileEnv)
	} else {
		log.WithField("service", serviceName).Info("Because we are running in development mode, the service won't be stopped")
	}

	if _, err := os.Stat(sats.AgentConfigFilePath); err == nil {
		os.Remove(sats.AgentConfigFilePath)
		log.WithFields(log.Fields{
			"path": sats.AgentConfigFilePath,
		}).Debug("Elastic Agent configuration file removed.")
	}
}

func (sats *StandAloneTestSuite) contributeSteps(s *godog.Suite) {
	s.Step(`^a "([^"]*)" stand-alone agent is deployed$`, sats.aStandaloneAgentIsDeployed)
	s.Step(`^there is new data in the index from agent$`, sats.thereIsNewDataInTheIndexFromAgent)
	s.Step(`^the "([^"]*)" docker container is stopped$`, sats.theDockerContainerIsStopped)
	s.Step(`^there is no new data in the index after agent shuts down$`, sats.thereIsNoNewDataInTheIndexAfterAgentShutsDown)
}

func (sats *StandAloneTestSuite) aStandaloneAgentIsDeployed(image string) error {
	log.Trace("Deploying an agent to Fleet")

	serviceManager := services.NewServiceManager()

	profileEnv["elasticAgentDockerImageSuffix"] = ""
	if image != "default" {
		profileEnv["elasticAgentDockerImageSuffix"] = "-" + image
	}

	containerName := fmt.Sprintf("%s_%s_%d", FleetProfileName, ElasticAgentServiceName, 1)

	configurationFileURL := "https://raw.githubusercontent.com/elastic/beats/master/x-pack/elastic-agent/elastic-agent.docker.yml"

	configurationFilePath, err := e2e.DownloadFile(configurationFileURL)
	if err != nil {
		return err
	}
	sats.AgentConfigFilePath = configurationFilePath

	profileEnv["elasticAgentContainerName"] = containerName
	profileEnv["elasticAgentConfigFile"] = sats.AgentConfigFilePath
	profileEnv["elasticAgentTag"] = agentVersion

	err = serviceManager.AddServicesToCompose(FleetProfileName, []string{ElasticAgentServiceName}, profileEnv)
	if err != nil {
		log.Error("Could not deploy the elastic-agent")
		return err
	}

	// get container hostname once
	hostname, err := getContainerHostname(containerName)
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
	serviceManager := services.NewServiceManager()

	profile := FleetProfileName
	serviceName := ElasticAgentServiceName

	composes := []string{
		profile,     // profile name
		serviceName, // agent service
	}
	err := serviceManager.RunCommand(profile, composes, []string{"logs", serviceName}, profileEnv)
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
	maxTimeout := time.Duration(timeoutFactor) * time.Minute * 2
	minimumHitsCount := 50

	result, err := searchAgentData(sats.Hostname, sats.RuntimeDependenciesStartDate, minimumHitsCount, maxTimeout)
	if err != nil {
		return err
	}

	log.Tracef("Search result: %v", result)

	return e2e.AssertHitsArePresent(result)
}

func (sats *StandAloneTestSuite) theDockerContainerIsStopped(serviceName string) error {
	serviceManager := services.NewServiceManager()

	err := serviceManager.RemoveServicesFromCompose(FleetProfileName, []string{serviceName}, profileEnv)
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

	return e2e.AssertHitsAreNotPresent(result)
}

func searchAgentData(hostname string, startDate time.Time, minimumHitsCount int, maxTimeout time.Duration) (e2e.SearchResult, error) {
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

	result, err := e2e.WaitForNumberOfHits(indexName, esQuery, minimumHitsCount, maxTimeout)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn(e2e.WaitForIndices())
	}

	return result, err
}
