// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/docker"
	"github.com/elastic/e2e-testing/internal/installer"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"

	"github.com/elastic/e2e-testing/internal/elasticsearch"
	log "github.com/sirupsen/logrus"
)

func (fts *FleetTestSuite) aStandaloneAgentIsDeployed(image string) error {
	return fts.startStandAloneAgent(image, "", nil)
}

func (fts *FleetTestSuite) bootstrapFleetServerFromAStandaloneAgent(image string) error {
	fleetPolicy, err := fts.kibanaClient.GetDefaultPolicy(true)
	if err != nil {
		return err
	}

	fts.FleetServerPolicy = fleetPolicy
	return fts.startStandAloneAgent(image, "", map[string]string{"fleetServerMode": "1"})
}

func (fts *FleetTestSuite) aStandaloneAgentIsDeployedWithFleetServerModeOnCloud(image string) error {
	fleetPolicy, err := fts.kibanaClient.GetDefaultPolicy(true)
	if err != nil {
		return err
	}
	fts.FleetServerPolicy = fleetPolicy
	volume := path.Join(config.OpDir(), "compose", "services", "elastic-agent", "apm-legacy")
	return fts.startStandAloneAgent(image, "docker-compose-cloud.yml", map[string]string{"apmVolume": volume})
}

func (fts *FleetTestSuite) thereIsNewDataInTheIndexFromAgent() error {
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	minimumHitsCount := 50

	result, err := searchAgentData(fts.Hostname, fts.RuntimeDependenciesStartDate, minimumHitsCount, maxTimeout)
	if err != nil {
		return err
	}

	log.Tracef("Search result: %v", result)

	return elasticsearch.AssertHitsArePresent(result)
}

func (fts *FleetTestSuite) theDockerContainerIsStopped(serviceName string) error {
	err := fts.deployer.Remove([]string{common.FleetProfileName, serviceName}, common.ProfileEnv)
	if err != nil {
		return err
	}
	fts.AgentStoppedDate = time.Now().UTC()

	return nil
}

func (fts *FleetTestSuite) thereIsNoNewDataInTheIndexAfterAgentShutsDown() error {
	maxTimeout := time.Duration(30) * time.Second
	minimumHitsCount := 1

	result, err := searchAgentData(fts.Hostname, fts.AgentStoppedDate, minimumHitsCount, maxTimeout)
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

func (fts *FleetTestSuite) startStandAloneAgent(image string, composeFilename string, env map[string]string) error {
	fts.StandAlone = true
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

	common.ProfileEnv["elasticAgentDockerImageSuffix"] = ""
	if image != "default" {
		common.ProfileEnv["elasticAgentDockerImageSuffix"] = "-" + image
	}

	common.ProfileEnv["elasticAgentDockerNamespace"] = utils.GetDockerNamespaceEnvVar("beats")

	containerName := fmt.Sprintf("%s_%s_%d", common.FleetProfileName, common.ElasticAgentServiceName, 1)

	common.ProfileEnv["elasticAgentContainerName"] = containerName
	common.ProfileEnv["elasticAgentPlatform"] = "linux/amd64"
	common.ProfileEnv["elasticAgentTag"] = dockerImageTag

	for k, v := range env {
		common.ProfileEnv[k] = v
	}

	services := []string{common.FleetProfileName, common.ElasticAgentServiceName}
	err := fts.deployer.Add(services, common.ProfileEnv)
	if err != nil {
		log.Error("Could not deploy the elastic-agent")
		return err
	}

	// get container hostname once
	hostname, err := docker.GetContainerHostname(containerName)
	if err != nil {
		return err
	}

	fts.Image = image
	fts.Hostname = hostname

	err = fts.installTestTools(containerName)
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) thePolicyShowsTheDatasourceAdded(packageName string) error {
	log.WithFields(log.Fields{
		"policyID": fts.Policy.ID,
		"package":  packageName,
	}).Trace("Checking if the policy shows the package added")

	maxTimeout := time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	configurationIsPresentFn := func() error {
		packagePolicy, err := fts.kibanaClient.GetIntegrationFromAgentPolicy(packageName, fts.Policy)
		if err != nil {
			log.WithFields(log.Fields{
				"packagePolicy": packagePolicy,
				"policy":        fts.Policy,
				"retry":         retryCount,
				"error":         err,
			}).Warn("The integration was not found in the policy")
			retryCount++
			return err
		}

		retryCount++
		return err
	}

	err := backoff.Retry(configurationIsPresentFn, exp)
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
