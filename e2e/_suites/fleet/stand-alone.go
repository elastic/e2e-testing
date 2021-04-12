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
	shell "github.com/elastic/e2e-testing/cli/shell"
	"github.com/elastic/e2e-testing/e2e"
	"github.com/elastic/e2e-testing/e2e/steps"
	"github.com/elastic/e2e-testing/internal/compose"
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
	serviceManager := compose.NewServiceManager()
	serviceName := common.ElasticAgentServiceName

	if log.IsLevelEnabled(log.DebugLevel) {
		_ = sats.getContainerLogs()
	}

	if !developerMode {
		_ = serviceManager.RemoveServicesFromCompose(context.Background(), common.FleetProfileName, []string{serviceName}, common.ProfileEnv)
	} else {
		log.WithField("service", serviceName).Info("Because we are running in development mode, the service won't be stopped")
	}

	if _, err := os.Stat(sats.AgentConfigFilePath); err == nil {
		beatsLocalPath := shell.GetEnv("BEATS_LOCAL_PATH", "")
		if beatsLocalPath == "" {
			os.Remove(sats.AgentConfigFilePath)

			log.WithFields(log.Fields{
				"path": sats.AgentConfigFilePath,
			}).Trace("Elastic Agent configuration file removed.")
		} else {
			log.WithFields(log.Fields{
				"path": sats.AgentConfigFilePath,
			}).Trace("Elastic Agent configuration file not removed because it's part of a repository.")
		}
	}
}

func (sats *StandAloneTestSuite) contributeSteps(s *godog.ScenarioContext) {
	s.Step(`^a "([^"]*)" stand-alone agent is deployed$`, sats.aStandaloneAgentIsDeployed)
	s.Step(`^a "([^"]*)" stand-alone agent is deployed with fleet server mode$`, sats.aStandaloneAgentIsDeployedWithFleetServerMode)
	s.Step(`^there is new data in the index from agent$`, sats.thereIsNewDataInTheIndexFromAgent)
	s.Step(`^the "([^"]*)" docker container is stopped$`, sats.theDockerContainerIsStopped)
	s.Step(`^there is no new data in the index after agent shuts down$`, sats.thereIsNoNewDataInTheIndexAfterAgentShutsDown)
	s.Step(`^the agent is listed in Fleet as "([^"]*)"$`, sats.theAgentIsListedInFleetWithStatus)
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

	dockerImageTag := agentVersion

	useCISnapshots := shell.GetEnvBool("BEATS_USE_CI_SNAPSHOTS")
	beatsLocalPath := shell.GetEnv("BEATS_LOCAL_PATH", "")
	if useCISnapshots || beatsLocalPath != "" {
		// load the docker images that were already:
		// a. downloaded from the GCP bucket
		// b. fetched from the local beats binaries
		dockerInstaller := GetElasticAgentInstaller("docker", image, agentVersion)

		dockerInstaller.PreInstallFn()

		dockerImageTag += "-amd64"
	}

	configurationFilePath, err := steps.FetchBeatConfiguration(true, "elastic-agent", "elastic-agent.docker.yml")
	if err != nil {
		return err
	}

	serviceManager := compose.NewServiceManager()

	profileEnv["elasticAgentDockerImageSuffix"] = ""
	if image != "default" {
		profileEnv["elasticAgentDockerImageSuffix"] = "-" + image
	}

	profileEnv["elasticAgentDockerNamespace"] = e2e.GetDockerNamespaceEnvVar()

	containerName := fmt.Sprintf("%s_%s_%d", FleetProfileName, ElasticAgentServiceName, 1)

	sats.AgentConfigFilePath = configurationFilePath

	profileEnv["elasticAgentContainerName"] = containerName
	profileEnv["elasticAgentConfigFile"] = sats.AgentConfigFilePath
	profileEnv["elasticAgentPlatform"] = "linux/amd64"
	profileEnv["elasticAgentTag"] = dockerImageTag

	for k, v := range env {
		profileEnv[k] = v
	}

	err := serviceManager.AddServicesToCompose(context.Background(), common.FleetProfileName, []string{common.ElasticAgentServiceName}, common.ProfileEnv)
	if err != nil {
		log.Error("Could not deploy the elastic-agent")
		return err
	}

	// get container hostname once
	hostname, err := steps.GetContainerHostname(containerName)
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

	profile := FleetProfileName
	serviceName := ElasticAgentServiceName

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

	result, err := e2e.WaitForNumberOfHits(context.Background(), indexName, esQuery, minimumHitsCount, maxTimeout)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn(e2e.WaitForIndices())
	}

	return result, err
}
