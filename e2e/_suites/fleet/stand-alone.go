// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	elasticversion "github.com/elastic/e2e-testing/internal"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/installer"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/types"
	"github.com/elastic/e2e-testing/internal/utils"

	"github.com/elastic/e2e-testing/internal/elasticsearch"
	log "github.com/sirupsen/logrus"
)

func (fts *FleetTestSuite) aStandaloneAgentIsDeployed(image string) error {
	return fts.startStandAloneAgent(image, "", nil)
}

func (fts *FleetTestSuite) bootstrapFleetServerFromAStandaloneAgent(image string) error {
	return fts.startStandAloneAgent(image, "", map[string]string{"fleetServerMode": "1"})
}

func (fts *FleetTestSuite) thereIsNewDataInTheIndexFromAgent() error {
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	minimumHitsCount := 20

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName).WithFlavour(fts.Image)

	manifest, _ := fts.deployer.Inspect(fts.currentContext, agentService)
	result, err := searchAgentData(fts.currentContext, manifest.Hostname, fts.RuntimeDependenciesStartDate, minimumHitsCount, maxTimeout)
	if err != nil {
		return err
	}

	log.Tracef("Search result: %v", result)

	return elasticsearch.AssertHitsArePresent(result)
}

func (fts *FleetTestSuite) theDockerContainerIsStopped(serviceName string) error {
	agentService := deploy.NewServiceRequest(serviceName)
	err := fts.deployer.Stop(fts.currentContext, agentService)
	if err != nil {
		return err
	}
	fts.AgentStoppedDate = time.Now().UTC()

	return nil
}

func (fts *FleetTestSuite) thereIsNoNewDataInTheIndexAfterAgentShutsDown() error {
	maxTimeout := time.Duration(30) * time.Second
	minimumHitsCount := 1

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.deployer.Inspect(fts.currentContext, agentService)
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

func (fts *FleetTestSuite) startStandAloneAgent(image string, flavour string, env map[string]string) error {
	fts.StandAlone = true
	log.Trace("Deploying an agent to Fleet")

	dockerImageTag := common.BeatVersion

	common.ProfileEnv["elasticAgentDockerNamespace"] = deploy.GetDockerNamespaceEnvVar("beats")
	common.ProfileEnv["elasticAgentDockerImageSuffix"] = ""
	if image != "default" {
		common.ProfileEnv["elasticAgentDockerImageSuffix"] = "-" + image
	}

	useCISnapshots := elasticversion.GithubCommitSha1 != ""

	if useCISnapshots || elasticversion.BeatsLocalPath != "" {
		// load the docker images that were already:
		// a. downloaded from the GCP bucket
		// b. fetched from the local beats binaries
		agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
		dockerInstaller, _ := installer.NewElasticAgentDeployer(fts.deployer).AttachInstaller(fts.currentContext, agentService, "docker")
		dockerInstaller.Preinstall(fts.currentContext)

		arch := types.Architectures[types.GetArchitecture()]
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

	for k, v := range env {
		common.ProfileEnv[k] = v
	}

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName).WithFlavour(flavour)

	err = fts.deployer.Add(fts.currentContext, deploy.NewServiceRequest(common.FleetProfileName), []deploy.ServiceRequest{agentService}, common.ProfileEnv)
	if err != nil {
		log.Error("Could not deploy the elastic-agent")
		return err
	}

	fts.Image = image

	manifest, _ := fts.deployer.Inspect(fts.currentContext, agentService)

	err = fts.installTestTools(manifest.Name)
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
		packagePolicy, err := fts.kibanaClient.GetIntegrationFromAgentPolicy(fts.currentContext, packageName, fts.Policy)
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

func searchAgentData(ctx context.Context, hostname string, startDate time.Time, minimumHitsCount int, maxTimeout time.Duration) (elasticsearch.SearchResult, error) {
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

	result, err := elasticsearch.WaitForNumberOfHits(ctx, indexName, esQuery, minimumHitsCount, maxTimeout)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn(elasticsearch.WaitForIndices())
	}

	return result, err
}
