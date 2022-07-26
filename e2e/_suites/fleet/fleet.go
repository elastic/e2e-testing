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
	"runtime"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"go.elastic.co/apm"

	"github.com/cenkalti/backoff/v4"
	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/elasticsearch"
	"github.com/elastic/e2e-testing/internal/installer"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"

	"github.com/pkg/errors"
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

func (fts *FleetTestSuite) anStaleAgentIsDeployedToFleetWithInstaller(staleVersion string, installerType string) error {
	switch staleVersion {
	case "latest":
		staleVersion = common.ElasticAgentVersion
	}

	fts.Version = staleVersion

	log.Tracef("The stale version is %s", fts.Version)

	return fts.anAgentIsDeployedToFleetWithInstaller(installerType)
}

// this step infers the installer type from the underlying OS image
// supported images: centos and debian
func (fts *FleetTestSuite) anAgentIsDeployedToFleet(image string) error {
	installerType := "rpm"
	if image == "debian" {
		installerType = "deb"
	}
	fts.BeatsProcess = ""

	// FIXME: We need to cleanup the steps to support different operating systems
	// for now we will force the zip installer type when the agent is running on windows
	if runtime.GOOS == "windows" && common.Provider == "remote" {
		installerType = "zip"
	}
	return fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(installerType)
}

func (fts *FleetTestSuite) anAgentIsDeployedToFleetOnTopOfBeat(beatsProcess string) error {
	installerType := "tar"

	// FIXME: We need to cleanup the steps to support different operating systems
	// for now we will force the zip installer type when the agent is running on windows
	if runtime.GOOS == "windows" && common.Provider == "remote" {
		installerType = "zip"
	}

	fts.BeatsProcess = beatsProcess

	return fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(installerType)
}

// supported installers: tar, rpm, deb
func (fts *FleetTestSuite) anAgentIsDeployedToFleetWithInstaller(installerType string) error {
	fts.BeatsProcess = ""

	// FIXME: We need to cleanup the steps to support different operating systems
	// for now we will force the zip installer type when the agent is running on windows
	if runtime.GOOS == "windows" && common.Provider == "remote" {
		installerType = "zip"
	}

	return fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(installerType)
}

// supported installers: tar, rpm, deb
func (fts *FleetTestSuite) anAgentIsDeployedToFleetWithInstallerAndTags(installerType string, flags string) error {
	fts.BeatsProcess = ""

	// FIXME: We need to cleanup the steps to support different operating systems
	// for now we will force the zip installer type when the agent is running on windows
	if runtime.GOOS == "windows" && common.Provider == "remote" {
		installerType = "zip"
	}
	fts.ElasticAgentFlags = flags
	return fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(installerType)
}

func (fts *FleetTestSuite) tagsAreInTheElasticAgentIndex() error {
	var tagsArray []string
	//ex of flags  "--tag production,linux" or "--tag=production,linux"
	if fts.ElasticAgentFlags != "" {
		tags := strings.TrimPrefix(fts.ElasticAgentFlags, "--tag")
		tags = strings.TrimPrefix(tags, "=")
		tags = strings.ReplaceAll(tags, " ", "")
		tagsArray = strings.Split(tags, ",")
	}
	if len(tagsArray) == 0 {
		return errors.Errorf("no tags were found, ElasticAgentFlags value %s", fts.ElasticAgentFlags)
	}

	var tagTerms []map[string]interface{}
	for _, tag := range tagsArray {
		tagTerms = append(tagTerms, map[string]interface{}{
			"term": map[string]interface{}{
				"tags": tag,
			},
		})
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": tagTerms,
			},
		},
	}

	indexName := ".fleet-agents"

	_, err := elasticsearch.WaitForNumberOfHits(context.Background(), indexName, query, 1, 3*time.Minute)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn(elasticsearch.WaitForIndices())
	}
	return err
}

func (fts *FleetTestSuite) anAgentIsDeployedToFleetWithInstallerAndFleetServer(installerType string) error {
	log.WithFields(log.Fields{
		"installer": installerType,
	}).Trace("Deploying an agent to Fleet with base image using an already bootstrapped Fleet Server")

	deployedAgentsCount++

	fts.InstallerType = installerType

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName).
		WithScale(deployedAgentsCount).
		WithVersion(fts.Version)

	if fts.BeatsProcess != "" {
		agentService = agentService.WithBackgroundProcess(fts.BeatsProcess)
	}

	services := []deploy.ServiceRequest{
		agentService,
	}
	env := fts.getProfileEnv()
	err := fts.getDeployer().Add(fts.currentContext, deploy.NewServiceRequest(common.FleetProfileName), services, env)
	if err != nil {
		return err
	}

	agentInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, installerType)
	err = deployAgentToFleet(fts.currentContext, agentInstaller, fts.CurrentToken, fts.ElasticAgentFlags)
	if err != nil {
		return err
	}
	return err
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

func (fts *FleetTestSuite) theAgentIsListedInFleetWithStatus(desiredStatus string) error {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.getDeployer().Inspect(fts.currentContext, agentService)
	err := theAgentIsListedInFleetWithStatus(fts.currentContext, desiredStatus, manifest.Hostname)
	if err != nil {
		return err
	}
	if desiredStatus == "online" {
		//get Agent Default Key
		err := fts.theAgentGetDefaultAPIKey()
		if err != nil {
			return err
		}
	}
	return err
}

func (fts *FleetTestSuite) theAgentGetDefaultAPIKey() error {
	defaultAPIKey, _ := fts.getAgentDefaultAPIKey()
	log.WithFields(log.Fields{
		"default_api_key": defaultAPIKey,
	}).Info("The Agent is installed with Default Api Key")
	fts.DefaultAPIKey = defaultAPIKey
	return nil
}

func (fts *FleetTestSuite) verifyDefaultAPIKey(status string) error {
	newDefaultAPIKey, _ := fts.getAgentDefaultAPIKey()

	logFields := log.Fields{
		"new_default_api_key": newDefaultAPIKey,
		"old_default_api_key": fts.DefaultAPIKey,
	}

	defaultAPIKeyHasChanged := (newDefaultAPIKey != fts.DefaultAPIKey)

	if status == "changed" {
		if !defaultAPIKeyHasChanged {
			log.WithFields(logFields).Error("Integration added and Default API Key do not change")
			return errors.New("Integration added and Default API Key do not change")
		}

		log.WithFields(logFields).Infof("Default API Key has %s when the Integration has been added", status)
		return nil
	}

	if status == "not changed" {
		if defaultAPIKeyHasChanged {
			log.WithFields(logFields).Error("Integration updated and Default API Key is changed")
			return errors.New("Integration updated and Default API Key is changed")
		}

		log.WithFields(logFields).Infof("Default API Key has %s when the Integration has been updated", status)
		return nil
	}

	log.Warnf("Status %s is not supported yet", status)
	return godog.ErrPending
}

func theAgentIsListedInFleetWithStatus(ctx context.Context, desiredStatus string, hostname string) error {
	log.Tracef("Checking if agent is listed in Fleet as %s", desiredStatus)

	kibanaClient, err := kibana.NewClient()
	if err != nil {
		return err
	}
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	agentOnlineFn := func() error {
		agentID, err := kibanaClient.GetAgentIDByHostname(ctx, hostname)
		if err != nil {
			retryCount++
			return err
		}

		if agentID == "" {
			// the agent is not listed in Fleet
			if desiredStatus == "offline" || desiredStatus == "inactive" {
				log.WithFields(log.Fields{
					"elapsedTime": exp.GetElapsedTime(),
					"hostname":    hostname,
					"retries":     retryCount,
					"status":      desiredStatus,
				}).Info("The Agent is not present in Fleet, as expected")
				return nil
			}

			retryCount++
			return fmt.Errorf("the agent is not present in Fleet in the '%s' status, but it should", desiredStatus)
		}

		agentStatus, err := kibanaClient.GetAgentStatusByHostname(ctx, hostname)
		isAgentInStatus := strings.EqualFold(agentStatus, desiredStatus)
		if err != nil || !isAgentInStatus {
			if err == nil {
				err = fmt.Errorf("the Agent is not in the %s status yet", desiredStatus)
			}

			log.WithFields(log.Fields{
				"agentID":         agentID,
				"isAgentInStatus": isAgentInStatus,
				"elapsedTime":     exp.GetElapsedTime(),
				"hostname":        hostname,
				"retry":           retryCount,
				"status":          desiredStatus,
			}).Warn(err.Error())

			retryCount++

			return err
		}
		log.WithFields(log.Fields{
			"isAgentInStatus": isAgentInStatus,
			"elapsedTime":     exp.GetElapsedTime(),
			"hostname":        hostname,
			"retries":         retryCount,
			"status":          desiredStatus,
		}).Info("The Agent is in the desired status")
		return nil
	}

	err = backoff.Retry(agentOnlineFn, exp)
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) theFileSystemAgentFolderIsEmpty() error {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	agentInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, fts.InstallerType)

	pkgManifest, _ := agentInstaller.Inspect()
	cmd := []string{
		"ls", "-l", pkgManifest.WorkDir,
	}

	content, err := agentInstaller.Exec(fts.currentContext, cmd)
	if err != nil {
		if content == "" || strings.Contains(content, "No such file or directory") {
			return nil
		}
		return err
	}

	log.WithFields(log.Fields{
		"installer":  agentInstaller,
		"workingDir": pkgManifest.WorkDir,
		"content":    content,
	}).Debug("Agent working dir content")

	return fmt.Errorf("the file system directory is not empty")
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

func (fts *FleetTestSuite) systemPackageDashboardsAreListedInFleet() error {
	log.Trace("Checking system Package dashboards in Fleet")

	dataStreamsCount := 0
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	countDataStreamsFn := func() error {
		dataStreams, err := fts.kibanaClient.GetDataStreams(fts.currentContext)
		if err != nil {
			log.WithFields(log.Fields{
				"retry":       retryCount,
				"elapsedTime": exp.GetElapsedTime(),
			}).Warn(err.Error())

			retryCount++

			return err
		}

		count := len(dataStreams.Children())
		if count == 0 {
			err = fmt.Errorf("there are no datastreams yet")

			log.WithFields(log.Fields{
				"retry":       retryCount,
				"dataStreams": count,
				"elapsedTime": exp.GetElapsedTime(),
			}).Warn(err.Error())

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"elapsedTime": exp.GetElapsedTime(),
			"datastreams": count,
			"retries":     retryCount,
		}).Info("Datastreams are present")
		dataStreamsCount = count
		return nil
	}

	err := backoff.Retry(countDataStreamsFn, exp)
	if err != nil {
		return err
	}

	if dataStreamsCount == 0 {
		err = fmt.Errorf("there are no datastreams. We expected to have more than one")
		log.Error(err.Error())
		return err
	}

	return nil
}

func (fts *FleetTestSuite) theAgentIsUnenrolled() error {
	return fts.unenrollHostname()
}

func (fts *FleetTestSuite) theAgentIsReenrolledOnTheHost() error {
	log.Trace("Re-enrolling the agent on the host with same token")

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	agentInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, fts.InstallerType)

	err := agentInstaller.Enroll(fts.currentContext, fts.CurrentToken, fts.ElasticAgentFlags)
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) theEnrollmentTokenIsRevoked() error {
	log.WithFields(log.Fields{
		"token":   fts.CurrentToken,
		"tokenID": fts.CurrentTokenID,
	}).Trace("Revoking enrollment token")

	err := fts.kibanaClient.DeleteEnrollmentAPIKey(fts.currentContext, fts.CurrentTokenID)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"token":   fts.CurrentToken,
		"tokenID": fts.CurrentTokenID,
	}).Debug("Token was revoked")

	// FIXME: Remove once https://github.com/elastic/kibana/issues/105078 is addressed
	utils.Sleep(time.Duration(utils.TimeoutFactor) * 20 * time.Second)
	return nil
}

// theVersionOfThePackageIsInstalled installs a package in a version
func (fts *FleetTestSuite) theVersionOfThePackageIsInstalled(version string, packageName string) error {
	log.WithFields(log.Fields{
		"package": packageName,
		"version": version,
	}).Trace("Checking if package version is installed")

	integration, err := fts.kibanaClient.GetIntegrationByPackageName(fts.currentContext, packageName)
	if err != nil {
		return err
	}

	_, err = fts.kibanaClient.InstallIntegrationAssets(fts.currentContext, integration)
	if err != nil {
		return err
	}
	fts.Integration = integration

	return nil
}

func (fts *FleetTestSuite) anAttemptToEnrollANewAgentFails() error {
	log.Trace("Enrolling a new agent with an revoked token")

	// increase the number of agents
	deployedAgentsCount++

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName).WithScale(deployedAgentsCount)
	services := []deploy.ServiceRequest{
		agentService,
	}
	env := fts.getProfileEnv()
	err := fts.getDeployer().Add(fts.currentContext, deploy.NewServiceRequest(common.FleetProfileName), services, env)
	if err != nil {
		return err
	}

	agentInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, fts.InstallerType)
	err = deployAgentToFleet(fts.currentContext, agentInstaller, fts.CurrentToken, fts.ElasticAgentFlags)

	if err == nil {
		err = fmt.Errorf("the agent was enrolled although the token was previously revoked")

		log.WithFields(log.Fields{
			"tokenID": fts.CurrentTokenID,
			"error":   err,
		}).Error(err.Error())
		return err
	}

	// checking the error message produced by the install command in TAR installer
	// to distinguish from other install errors
	if err != nil && strings.Contains(err.Error(), "Error: enroll command failed") {
		log.WithFields(log.Fields{
			"err":   err,
			"token": fts.CurrentToken,
		}).Debug("As expected, it's not possible to enroll an agent with a revoked token")
		return nil
	}

	return nil
}

// unenrollHostname deletes the statuses for an existing agent, filtering by hostname
func (fts *FleetTestSuite) unenrollHostname() error {
	span, _ := apm.StartSpanOptions(fts.currentContext, "Unenrolling hostname", "elastic-agent.hostname.unenroll", apm.SpanOptions{
		Parent: apm.SpanFromContext(fts.currentContext).TraceContext(),
	})
	defer span.End()

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.getDeployer().Inspect(fts.currentContext, agentService)
	log.Tracef("Un-enrolling all agentIDs for %s", manifest.Hostname)

	agents, err := fts.kibanaClient.ListAgents(fts.currentContext)
	if err != nil {
		return err
	}

	for _, agent := range agents {
		if agent.LocalMetadata.Host.HostName == manifest.Hostname {
			log.WithFields(log.Fields{
				"hostname": manifest.Hostname,
			}).Debug("Un-enrolling agent in Fleet")

			err := fts.kibanaClient.UnEnrollAgent(fts.currentContext, agent.LocalMetadata.Host.HostName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (fts *FleetTestSuite) checkDataStream() error {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []interface{}{
					map[string]interface{}{
						"exists": map[string]interface{}{
							"field": "linux.memory.page_stats",
						},
					},
					map[string]interface{}{
						"exists": map[string]interface{}{
							"field": "elastic_agent",
						},
					},
					map[string]interface{}{
						"range": map[string]interface{}{
							"@timestamp": map[string]interface{}{
								"gte": "now-1m",
							},
						},
					},
					map[string]interface{}{
						"term": map[string]interface{}{
							"data_stream.type": "metrics",
						},
					},
					map[string]interface{}{
						"term": map[string]interface{}{
							"data_stream.dataset": "linux.memory",
						},
					},
					map[string]interface{}{
						"term": map[string]interface{}{
							"data_stream.namespace": "default",
						},
					},
					map[string]interface{}{
						"term": map[string]interface{}{
							"event.dataset": "linux.memory",
						},
					},
					map[string]interface{}{
						"term": map[string]interface{}{
							"agent.type": "metricbeat",
						},
					},
					map[string]interface{}{
						"term": map[string]interface{}{
							"metricset.period": 1000,
						},
					},
					map[string]interface{}{
						"term": map[string]interface{}{
							"service.type": "linux",
						},
					},
				},
			},
		},
	}

	indexName := "metrics-linux.memory-default"

	_, err := elasticsearch.WaitForNumberOfHits(context.Background(), indexName, query, 1, 3*time.Minute)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn(elasticsearch.WaitForIndices())
	}

	return err
}

func deployAgentToFleet(ctx context.Context, agentInstaller deploy.ServiceOperator, token string, flags string) error {
	err := agentInstaller.Preinstall(ctx)
	if err != nil {
		return err
	}

	err = agentInstaller.Install(ctx)
	if err != nil {
		return err
	}

	err = agentInstaller.Enroll(ctx, token, flags)
	if err != nil {
		return err
	}

	return agentInstaller.Postinstall(ctx)
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

func (fts *FleetTestSuite) getAgentDefaultAPIKey() (string, error) {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.getDeployer().Inspect(fts.currentContext, agentService)
	agent, err := fts.kibanaClient.GetAgentByHostnameFromList(fts.currentContext, manifest.Hostname)
	if err != nil {
		return "", err
	}
	return agent.DefaultAPIKey, nil
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
