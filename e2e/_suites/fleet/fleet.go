// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/elasticsearch"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"

	log "github.com/sirupsen/logrus"
)

const actionADDED = "added"
const actionREMOVED = "removed"
const testResourcesDir = "./testresources"

var deployedAgentsCount = 0

// FleetTestSuite represents the scenarios for Fleet-mode
type FleetTestSuite struct {
	// integrations
	KibanaProfile       string
	StandAlone          bool
	CurrentToken        string // current enrollment token
	CurrentTokenID      string // current enrollment tokenID
	ElasticAgentStopped bool   // will be used to signal when the agent process can be called again in the tear-down stage
	Image               string // base image used to install the agent
	InstallerType       string
	Integration         kibana.IntegrationPackage // the installed integration
	Policy              kibana.Policy
	PolicyUpdatedAt     string // the moment the policy was updated
	Version             string // current elastic-agent version
	kibanaClient        *kibana.Client
	deployer            deploy.Deployment
	dockerDeployer      deploy.Deployment // used for docker related deployents, such as the stand-alone containers
	BeatsProcess        string            // (optional) name of the Beats that must be present before installing the elastic-agent
	// date controls for queries
	AgentStoppedDate             time.Time
	RuntimeDependenciesStartDate time.Time
	// instrumentation
	currentContext    context.Context
	DefaultAPIKey     string
	ElasticAgentFlags string
}

func (fts *FleetTestSuite) getDeployer() deploy.Deployment {
	if fts.StandAlone {
		return fts.dockerDeployer
	}
	return fts.deployer
}

// bootstrapFleet this method creates the runtime dependencies for the Fleet test suite, being of special
// interest kibana profile passed as part of the environment variables to bootstrap the dependencies.
func bootstrapFleet(ctx context.Context, env map[string]string) error {
	deployer := deploy.New(common.Provider)

	if profile, ok := env["kibanaProfile"]; ok {
		log.Infof("Running kibana with %s profile", profile)
	}

	// the runtime dependencies must be started only in non-remote executions
	return deployer.Bootstrap(ctx, deploy.NewServiceRequest(common.FleetProfileName), env, func() error {
		kibanaClient, err := kibana.NewClient()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"env":   env,
			}).Fatal("Unable to create kibana client")
		}

		err = elasticsearch.WaitForClusterHealth(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Elasticsearch Cluster is not healthy")
		}

		err = kibanaClient.RecreateFleet(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"env":   env,
			}).Fatal("Fleet could not be recreated")
		}

		fleetServicePolicy := kibana.FleetServicePolicy

		log.WithFields(log.Fields{
			"id":          fleetServicePolicy.ID,
			"name":        fleetServicePolicy.Name,
			"description": fleetServicePolicy.Description,
		}).Info("Fleet Server Policy retrieved")

		maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
		exp := utils.GetExponentialBackOff(maxTimeout)
		retryCount := 1

		fleetServerBootstrapFn := func() error {
			serviceToken, err := elasticsearch.GetAPIToken(ctx)
			if err != nil {
				retryCount++
				log.WithFields(log.Fields{
					"error":       err,
					"retries":     retryCount,
					"elapsedTime": exp.GetElapsedTime(),
				}).Warn("Could not get API Token from Elasticsearch.")
				return err
			}

			fleetServerEnv := make(map[string]string)
			for k, v := range env {
				fleetServerEnv[k] = v
			}

			fleetServerPort, err := nat.NewPort("tcp", "8220")
			if err != nil {
				retryCount++
				log.WithFields(log.Fields{
					"error":       err,
					"retries":     retryCount,
					"elapsedTime": exp.GetElapsedTime(),
				}).Warn("Could not create TCP port for fleet-server")
				return err
			}

			fleetServerEnv["elasticAgentTag"] = common.ElasticAgentVersion
			fleetServerEnv["fleetServerMode"] = "1"
			fleetServerEnv["fleetServerPort"] = fleetServerPort.Port()
			fleetServerEnv["fleetInsecure"] = "1"
			fleetServerEnv["fleetServerServiceToken"] = serviceToken.AccessToken
			fleetServerEnv["fleetServerPolicyId"] = fleetServicePolicy.ID

			fleetServerSrv := deploy.ServiceRequest{
				Name:    common.ElasticAgentServiceName,
				Flavour: "fleet-server",
			}

			err = deployer.Add(ctx, deploy.NewServiceRequest(common.FleetProfileName), []deploy.ServiceRequest{fleetServerSrv}, fleetServerEnv)
			if err != nil {
				retryCount++
				log.WithFields(log.Fields{
					"error":       err,
					"retries":     retryCount,
					"elapsedTime": exp.GetElapsedTime(),
					"env":         fleetServerEnv,
				}).Warn("Fleet Server could not be started. Retrying")
				return err
			}

			log.WithFields(log.Fields{
				"retries":     retryCount,
				"elapsedTime": exp.GetElapsedTime(),
			}).Info("Fleet Server was started")
			return nil
		}

		err = backoff.Retry(fleetServerBootstrapFn, exp)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Fleet Server could not be started")
		}

		err = kibanaClient.WaitForFleet(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"env":   env,
			}).Fatal("Fleet could not be initialized")
		}
		return nil
	})
}

func (fts *FleetTestSuite) getProfileEnv() map[string]string {

	env := map[string]string{}

	for k, v := range common.ProfileEnv {
		env[k] = v
	}

	if fts.KibanaProfile != "" {
		env["kibanaProfile"] = fts.KibanaProfile
	}

	return env
}

func (fts *FleetTestSuite) setup() error {
	log.Trace("Creating Fleet setup")

	err := fts.kibanaClient.RecreateFleet(fts.currentContext)
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) theHostIsRestarted() error {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	err := fts.getDeployer().Stop(fts.currentContext, agentService)
	if err != nil {
		log.WithField("err", err).Error("Could not stop the service")
	}

	utils.Sleep(time.Duration(utils.TimeoutFactor) * 10 * time.Second)

	err = fts.getDeployer().Start(fts.currentContext, agentService)
	if err != nil {
		log.WithField("err", err).Error("Could not start the service")
	}

	log.Debug("The elastic-agent service has been restarted")
	return nil
}

func inputs(integration string) []kibana.Input {
	switch integration {
	case "apm":
		return []kibana.Input{
			{
				Type:    "apm",
				Enabled: true,
				Streams: []kibana.Stream{},
				Vars: map[string]kibana.Var{
					"apm-server": {
						Value: "host",
						Type:  "localhost:8200",
					},
				},
			},
		}
	case "linux":
		return []kibana.Input{
			{
				Type:    "linux/metrics",
				Enabled: true,
				Streams: []kibana.Stream{
					{
						ID:      "linux/metrics-linux.memory-" + uuid.New().String(),
						Enabled: true,
						DS: kibana.DataStream{
							Dataset: "linux.memory",
							Type:    "metrics",
						},
						Vars: map[string]kibana.Var{
							"period": {
								Value: "1s",
								Type:  "string",
							},
						},
					},
				},
			},
		}
	}
	return []kibana.Input{}
}

func metricsInputs(integration string, set string, file string, metrics string) []kibana.Input {
	metricsFile := filepath.Join(testResourcesDir, file)
	jsonData, err := readJSONFile(metricsFile)
	if err != nil {
		log.Warnf("An error happened while reading metrics file, returning an empty array of inputs: %v", err)
		return []kibana.Input{}
	}

	data := parseJSONMetrics(jsonData, integration, set, metrics)
	return []kibana.Input{
		{
			Type:    integration,
			Enabled: true,
			Streams: data,
		},
	}

	return []kibana.Input{}
}

func readJSONFile(file string) (*gabs.Container, error) {
	jsonFile, err := os.Open(file)
	if err != nil {
		fmt.Println(err)
	}
	log.WithFields(log.Fields{
		"file": file,
	}).Info("Successfully Opened " + file)

	defer jsonFile.Close()
	data, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	jsonParsed, err := gabs.ParseJSON(data)
	if err != nil {
		return nil, err
	}

	return jsonParsed.S("inputs"), nil
}

func parseJSONMetrics(data *gabs.Container, integration string, set string, metrics string) []kibana.Stream {
	for i, item := range data.Children() {
		if item.Path("type").Data().(string) == integration {
			for idx, stream := range item.S("streams").Children() {
				dataSet, _ := stream.Path("data_stream.dataset").Data().(string)
				if dataSet == metrics+"."+set {
					data.SetP(
						integration+"-"+metrics+"."+set+"-"+uuid.New().String(),
						fmt.Sprintf("inputs.%d.streams.%d.id", i, idx),
					)
					data.SetP(
						true,
						fmt.Sprintf("inputs.%d.streams.%d.enabled", i, idx),
					)

					var dataStreamOut []kibana.Stream
					if err := json.Unmarshal(data.Path(fmt.Sprintf("inputs.%d.streams", i)).Bytes(), &dataStreamOut); err != nil {
						return []kibana.Stream{}
					}

					return dataStreamOut
				}
			}
		}
	}
	return []kibana.Stream{}
}
