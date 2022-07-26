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
	"github.com/elastic/e2e-testing/pkg/downloads"

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

// afterScenario destroys the state created by a scenario
func (fts *FleetTestSuite) afterScenario() {
	defer func() {
		fts.DefaultAPIKey = ""
		// Reset Kibana Profile to default
		fts.KibanaProfile = ""
		deployedAgentsCount = 0
	}()

	span := tx.StartSpan("Clean up", "test.scenario.clean", nil)
	fts.currentContext = apm.ContextWithSpan(context.Background(), span)
	defer span.End()

	serviceName := common.ElasticAgentServiceName

	// if DEVELOPER_MODE=true let's not uninstall/unenroll the agent
	if !common.DeveloperMode && fts.InstallerType != "" {
		agentService := deploy.NewServiceRequest(serviceName)

		if !fts.StandAlone {
			// for the centos/debian flavour we need to retrieve the internal log files for the elastic-agent, as they are not
			// exposed as container logs. For that reason we need to go through the installer abstraction
			agentInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, fts.InstallerType)

			if log.IsLevelEnabled(log.DebugLevel) {
				err := agentInstaller.Logs(fts.currentContext)
				if err != nil {
					log.WithField("error", err).Warn("Could not get agent logs in the container")
				}
			}
			// only call it when the elastic-agent is present
			if !fts.ElasticAgentStopped {
				err := agentInstaller.Uninstall(fts.currentContext)
				if err != nil {
					log.Warnf("Could not uninstall the agent after the scenario: %v", err)
				}
			}
		} else if log.IsLevelEnabled(log.DebugLevel) {
			// for the Docker image, we simply retrieve container logs
			_ = fts.getDeployer().Logs(fts.currentContext, agentService)
		}

		err := fts.unenrollHostname()
		if err != nil {
			manifest, _ := fts.getDeployer().Inspect(fts.currentContext, agentService)
			log.WithFields(log.Fields{
				"err":      err,
				"hostname": manifest.Hostname,
			}).Warn("The agentIDs for the hostname could not be unenrolled")
		}
	}

	env := fts.getProfileEnv()
	_ = fts.getDeployer().Remove(fts.currentContext, deploy.NewServiceRequest(common.FleetProfileName), []deploy.ServiceRequest{deploy.NewServiceRequest(serviceName)}, env)

	// TODO: Determine why this may be empty here before being cleared out
	if fts.CurrentTokenID != "" {
		err := fts.kibanaClient.DeleteEnrollmentAPIKey(fts.currentContext, fts.CurrentTokenID)
		if err != nil {
			log.WithFields(log.Fields{
				"err":     err,
				"tokenID": fts.CurrentTokenID,
			}).Warn("The enrollment token could not be deleted")
		}
	}

	// TODO: Dont think this is needed if we are making all policies unique
	// fts.kibanaClient.DeleteAllPolicies(fts.currentContext)

	// clean up fields
	fts.CurrentTokenID = ""
	fts.CurrentToken = ""
	fts.InstallerType = ""
	fts.Image = ""
	fts.StandAlone = false
	fts.BeatsProcess = ""
	fts.ElasticAgentFlags = ""
}

// beforeScenario creates the state needed by a scenario
func (fts *FleetTestSuite) beforeScenario() {
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	exp := utils.GetExponentialBackOff(maxTimeout)

	fts.StandAlone = false
	fts.ElasticAgentStopped = false

	fts.Version = common.ElasticAgentVersion

	waitForPolicy := func() error {
		policy, err := fts.kibanaClient.CreatePolicy(fts.currentContext)
		if err != nil {
			return errors.Wrap(err, "A new policy could not be obtained, retrying.")
		}

		log.WithFields(log.Fields{
			"id":          policy.ID,
			"name":        policy.Name,
			"description": policy.Description,
		}).Info("Policy created")

		fts.Policy = policy

		// Grab the system integration as we'll need to assign it a new name so it wont collide during
		// multiple policy creations at once
		integration, err := fts.kibanaClient.GetIntegrationByPackageName(context.Background(), "system")
		if err != nil {
			return err
		}

		packageDataStream := kibana.PackageDataStream{
			Name:        fmt.Sprintf("%s-%s", integration.Name, uuid.New().String()),
			Description: integration.Title,
			Namespace:   "default",
			PolicyID:    fts.Policy.ID,
			Enabled:     true,
			Package:     integration,
			Inputs:      []kibana.Input{},
		}

		systemMetricsFile := filepath.Join(testResourcesDir, "/default_system_metrics.json")
		jsonData, err := readJSONFile(systemMetricsFile)
		if err != nil {
			return err
		}

		for _, item := range jsonData.Children() {
			var streams []kibana.Stream
			if err := json.Unmarshal(item.Path("streams").Bytes(), &streams); err != nil {
				log.WithError(err).Warn("Could not unmarshall streams, will use an empty array instead")
				streams = []kibana.Stream{}
			}

			if item.Path("type").Data().(string) == "system/metrics" {
				packageDataStream.Inputs = append(packageDataStream.Inputs, kibana.Input{
					Type:    item.Path("type").Data().(string),
					Enabled: item.Path("enabled").Data().(bool),
					Streams: streams,
					Vars: map[string]kibana.Var{
						"system.hostfs": {
							Value: "",
							Type:  "text",
						},
					},
				})
			} else {
				packageDataStream.Inputs = append(packageDataStream.Inputs, kibana.Input{
					Type:    item.Path("type").Data().(string),
					Enabled: item.Path("enabled").Data().(bool),
					Streams: streams,
				})
			}
		}

		err = fts.kibanaClient.AddIntegrationToPolicy(context.Background(), packageDataStream)
		if err != nil {
			return err
		}

		return nil
	}

	err := backoff.Retry(waitForPolicy, exp)
	if err != nil {
		log.Fatal(err)
	}

	// Grab a new enrollment key for new agent
	enrollmentKey, err := fts.kibanaClient.CreateEnrollmentAPIKey(fts.currentContext, fts.Policy)

	if err != nil {
		log.Fatal("Unable to create enrollment token for agent")
	}

	fts.CurrentToken = enrollmentKey.APIKey
	fts.CurrentTokenID = enrollmentKey.ID
}

func (fts *FleetTestSuite) contributeSteps(s *godog.ScenarioContext) {
	s.Step(`^kibana uses "([^"]*)" profile$`, fts.kibanaUsesProfile)
	s.Step(`^agent uses enrollment token from "([^"]*)" policy$`, fts.agentUsesPolicy)
	s.Step(`^a "([^"]*)" agent is deployed to Fleet$`, fts.anAgentIsDeployedToFleet)
	s.Step(`^an agent is deployed to Fleet on top of "([^"]*)"$`, fts.anAgentIsDeployedToFleetOnTopOfBeat)
	s.Step(`^an agent is deployed to Fleet with "([^"]*)" installer$`, fts.anAgentIsDeployedToFleetWithInstaller)
	s.Step(`^an agent is deployed to Fleet with "([^"]*)" installer and "([^"]*)" flags$`, fts.anAgentIsDeployedToFleetWithInstallerAndTags)
	s.Step(`^a "([^"]*)" stale agent is deployed to Fleet with "([^"]*)" installer$`, fts.anStaleAgentIsDeployedToFleetWithInstaller)
	s.Step(`^agent is in "([^"]*)" version$`, fts.agentInVersion)
	s.Step(`^agent is upgraded to "([^"]*)" version$`, fts.anAgentIsUpgradedToVersion)
	s.Step(`^the agent is listed in Fleet as "([^"]*)"$`, fts.theAgentIsListedInFleetWithStatus)
	s.Step(`^the default API key has "([^"]*)"$`, fts.verifyDefaultAPIKey)
	s.Step(`^the host is restarted$`, fts.theHostIsRestarted)
	s.Step(`^system package dashboards are listed in Fleet$`, fts.systemPackageDashboardsAreListedInFleet)
	s.Step(`^the agent is un-enrolled$`, fts.theAgentIsUnenrolled)
	s.Step(`^the agent is re-enrolled on the host$`, fts.theAgentIsReenrolledOnTheHost)
	s.Step(`^the enrollment token is revoked$`, fts.theEnrollmentTokenIsRevoked)
	s.Step(`^an attempt to enroll a new agent fails$`, fts.anAttemptToEnrollANewAgentFails)
	s.Step(`^the "([^"]*)" process is "([^"]*)" on the host$`, fts.processStateChangedOnTheHost)
	s.Step(`^the file system Agent folder is empty$`, fts.theFileSystemAgentFolderIsEmpty)
	s.Step(`^certs are installed$`, fts.installCerts)
	s.Step(`^a Linux data stream exists with some data$`, fts.checkDataStream)
	s.Step(`^the agent is enrolled into "([^"]*)" policy$`, fts.agentRunPolicy)

	//flags steps
	s.Step(`^the elastic agent index contains the tags$`, fts.tagsAreInTheElasticAgentIndex)

	// endpoint steps
	s.Step(`^the "([^"]*)" integration is "([^"]*)" in the policy$`, fts.theIntegrationIsOperatedInThePolicy)
	s.Step(`^the "([^"]*)" datasource is shown in the policy as added$`, fts.thePolicyShowsTheDatasourceAdded)
	s.Step(`^the host name is shown in the Administration view in the Security App as "([^"]*)"$`, fts.theHostNameIsShownInTheAdminViewInTheSecurityApp)
	s.Step(`^the host name is not shown in the Administration view in the Security App$`, fts.theHostNameIsNotShownInTheAdminViewInTheSecurityApp)
	s.Step(`^an "([^"]*)" is successfully deployed with an Agent using "([^"]*)" installer$`, fts.anIntegrationIsSuccessfullyDeployedWithAgentAndInstaller)
	s.Step(`^the policy response will be shown in the Security App$`, fts.thePolicyResponseWillBeShownInTheSecurityApp)
	s.Step(`^the policy is updated to have "([^"]*)" in "([^"]*)" mode$`, fts.thePolicyIsUpdatedToHaveMode)
	s.Step(`^the policy will reflect the change in the Security App$`, fts.thePolicyWillReflectTheChangeInTheSecurityApp)

	// System Integration steps
	s.Step(`^the policy is updated to have "([^"]*)" set to "([^"]*)"$`, fts.thePolicyIsUpdatedToHaveSystemSet)
	s.Step(`^"([^"]*)" with "([^"]*)" metrics are present in the datastreams$`, fts.theMetricsInTheDataStream)

	// stand-alone only steps
	s.Step(`^a "([^"]*)" stand-alone agent is deployed$`, fts.aStandaloneAgentIsDeployed)
	s.Step(`^a "([^"]*)" stand-alone agent is deployed with fleet server mode$`, fts.bootstrapFleetServerFromAStandaloneAgent)
	s.Step(`^there is new data in the index from agent$`, fts.thereIsNewDataInTheIndexFromAgent)
	s.Step(`^the "([^"]*)" docker container is stopped$`, fts.theDockerContainerIsStopped)
	s.Step(`^there is no new data in the index after agent shuts down$`, fts.thereIsNoNewDataInTheIndexAfterAgentShutsDown)
	s.Step(`^the stand-alone agent is listed in Fleet as "([^"]*)"$`, fts.theStandaloneAgentIsListedInFleetWithStatus)
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

	manifest, _ := fts.getDeployer().Inspect(fts.currentContext, agentService)
	return fts.kibanaClient.UpgradeAgent(fts.currentContext, manifest.Hostname, desiredVersion)
}

func (fts *FleetTestSuite) agentInVersion(version string) error {
	switch version {
	case "latest":
		version = downloads.GetSnapshotVersion(common.ElasticAgentVersion)
	}
	log.Tracef("Checking if agent is in version %s. Current version: %s", version, fts.Version)

	retryCount := 0
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	exp := utils.GetExponentialBackOff(maxTimeout)

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.getDeployer().Inspect(fts.currentContext, agentService)

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

func (fts *FleetTestSuite) agentRunPolicy(policyName string) error {
	agentRunPolicyFn := func() error {
		agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
		manifest, _ := fts.getDeployer().Inspect(fts.currentContext, agentService)

		policies, err := fts.kibanaClient.ListPolicies(fts.currentContext)
		if err != nil {
			return err
		}

		var policy *kibana.Policy
		for _, p := range policies {
			if policyName == p.Name {
				policy = &p
				break
			}
		}

		if policy == nil {
			return fmt.Errorf("Policy not found '%s'", policyName)
		}

		agent, err := fts.kibanaClient.GetAgentByHostnameFromList(fts.currentContext, manifest.Hostname)
		if err != nil {
			return err
		}

		if agent.PolicyID != policy.ID {
			log.Errorf("FOUND %s %s", agent.PolicyID, policy.ID)
			return fmt.Errorf("Agent not running the correct policy (running '%s' instead of '%s')", agent.PolicyID, policy.ID)
		}

		return nil
	}
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	exp := utils.GetExponentialBackOff(maxTimeout)

	return backoff.Retry(agentRunPolicyFn, exp)

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

// kibanaUsesProfile this step should be ideally called as a Background or a Given clause, so that it
// is executed before any other in the test scenario. It will configure the Kibana profile to be used
// in the scenario, changing the configuration file to be used.
func (fts *FleetTestSuite) kibanaUsesProfile(profile string) error {
	fts.KibanaProfile = profile

	env := fts.getProfileEnv()

	return bootstrapFleet(context.Background(), env)
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

func (fts *FleetTestSuite) agentUsesPolicy(policyName string) error {
	agentUsesPolicyFn := func() error {
		policies, err := fts.kibanaClient.ListPolicies(fts.currentContext)
		if err != nil {
			return err
		}

		for _, p := range policies {
			if policyName == p.Name {

				fts.Policy = p
				break
			}
		}

		if fts.Policy.Name != policyName {
			return fmt.Errorf("Policy not found '%s'", policyName)
		}

		return nil
	}
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	exp := utils.GetExponentialBackOff(maxTimeout)

	return backoff.Retry(agentUsesPolicyFn, exp)
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

func theIntegrationIsOperatedInThePolicy(ctx context.Context, client *kibana.Client, policy kibana.Policy, packageName string, action string) error {
	log.WithFields(log.Fields{
		"action":  action,
		"policy":  policy,
		"package": packageName,
	}).Trace("Doing an operation for a package on a policy")

	integration, err := client.GetIntegrationByPackageName(ctx, packageName)
	if err != nil {
		return err
	}

	if strings.ToLower(action) == actionADDED {
		packageDataStream := kibana.PackageDataStream{
			Name:        fmt.Sprintf("%s-%s", integration.Name, uuid.New().String()),
			Description: integration.Title,
			Namespace:   "default",
			PolicyID:    policy.ID,
			Enabled:     true,
			Package:     integration,
			Inputs:      []kibana.Input{},
		}
		packageDataStream.Inputs = inputs(integration.Name)

		err = client.AddIntegrationToPolicy(ctx, packageDataStream)
		if err != nil {
			log.WithFields(log.Fields{
				"err":       err,
				"packageDS": packageDataStream,
			}).Error("Unable to add integration to policy")
			return err
		}
	} else if strings.ToLower(action) == actionREMOVED {
		packageDataStream, err := client.GetIntegrationFromAgentPolicy(ctx, integration.Name, policy)
		if err != nil {
			return err
		}
		return client.DeleteIntegrationFromPolicy(ctx, packageDataStream)
	}

	return nil
}

func (fts *FleetTestSuite) anIntegrationIsSuccessfullyDeployedWithAgentAndInstaller(integration string, installerType string) error {
	err := fts.anAgentIsDeployedToFleetWithInstaller(installerType)
	if err != nil {
		return err
	}

	return fts.theIntegrationIsOperatedInThePolicy(integration, actionADDED)
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
