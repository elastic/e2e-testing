// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
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
	// date controls for queries
	AgentStoppedDate             time.Time
	RuntimeDependenciesStartDate time.Time
}

func (sats *StandAloneTestSuite) contributeSteps(s *godog.Suite) {
	s.Step(`^a stand-alone agent is deployed$`, sats.aStandaloneAgentIsDeployed)
	s.Step(`^there is new data in the index from agent$`, sats.thereIsNewDataInTheIndexFromAgent)
	s.Step(`^the "([^"]*)" docker container is stopped$`, sats.theDockerContainerIsStopped)
	s.Step(`^there is no new data in the index after agent shuts down$`, sats.thereIsNoNewDataInTheIndexAfterAgentShutsDown)
}

func (sats *StandAloneTestSuite) aStandaloneAgentIsDeployed() error {
	log.Debug("Deploying an agent to Fleet")

	serviceManager := services.NewServiceManager()

	profile := "ingest-manager"
	serviceName := "elastic-agent"

	configurationFileURL := "https://raw.githubusercontent.com/elastic/beats/master/x-pack/elastic-agent/elastic-agent.docker.yml"

	configurationFilePath, err := e2e.DownloadFile(configurationFileURL)
	if err != nil {
		return err
	}
	sats.AgentConfigFilePath = configurationFilePath

	profileEnv["elasticAgentConfigFile"] = sats.AgentConfigFilePath

	err = serviceManager.AddServicesToCompose(profile, []string{serviceName}, profileEnv)
	if err != nil {
		log.Error("Could not deploy the elastic-agent")
		return err
	}

	// get container hostname once
	hostname, err := getContainerHostname(serviceName)
	if err != nil {
		return err
	}

	sats.Hostname = hostname
	sats.Cleanup = true

	if log.IsLevelEnabled(log.DebugLevel) {
		composes := []string{
			profile,     // profile name
			serviceName, // agent service
		}
		err = serviceManager.RunCommand(profile, composes, []string{"logs", serviceName}, profileEnv)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"service": serviceName,
			}).Error("Could not retrieve Elastic Agent logs")

			return err
		}
	}

	return nil
}

func (sats *StandAloneTestSuite) thereIsNewDataInTheIndexFromAgent() error {
	maxTimeout := time.Duration(queryRetryTimeout) * time.Minute
	minimumHitsCount := 50

	result, err := searchAgentData(sats.Hostname, sats.RuntimeDependenciesStartDate, minimumHitsCount, maxTimeout)
	if err != nil {
		return err
	}

	log.Debugf("Search result: %v", result)

	return e2e.AssertHitsArePresent(result)
}

func (sats *StandAloneTestSuite) theDockerContainerIsStopped(serviceName string) error {
	serviceManager := services.NewServiceManager()

	err := serviceManager.RemoveServicesFromCompose("ingest-manager", []string{serviceName}, profileEnv)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"service": serviceName,
		}).Error("Could not stop the service.")

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
		log.WithFields(log.Fields{
			"error": err,
		}).Info("No documents were found for the Agent in the index after it stopped")
		return nil
	}

	return e2e.AssertHitsAreNotPresent(result)
}

// we need the container name because we use the Docker Client instead of Docker Compose
func getContainerHostname(serviceName string) (string, error) {
	containerName := "ingest-manager_" + serviceName + "_1"

	log.WithFields(log.Fields{
		"service":       serviceName,
		"containerName": containerName,
	}).Debug("Retrieving container name from the Docker client")

	hostname, err := docker.ExecCommandIntoContainer(context.Background(), containerName, "root", []string{"hostname"})
	if err != nil {
		log.WithFields(log.Fields{
			"containerName": containerName,
			"error":         err,
			"service":       serviceName,
		}).Error("Could not retrieve container name from the Docker client")
		return "", err
	}

	if strings.HasPrefix(hostname, "\x01\x00\x00\x00\x00\x00\x00\r") {
		hostname = strings.ReplaceAll(hostname, "\x01\x00\x00\x00\x00\x00\x00\r", "")
		log.WithFields(log.Fields{
			"hostname": hostname,
		}).Debug("Container name has been sanitized")
	}

	log.WithFields(log.Fields{
		"containerName": containerName,
		"hostname":      hostname,
		"service":       serviceName,
	}).Info("Hostname retrieved from the Docker client")

	return hostname, nil
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

	indexName := ".ds-logs-elastic.agent-default-000001"

	return e2e.WaitForNumberOfHits(indexName, esQuery, minimumHitsCount, maxTimeout)
}
