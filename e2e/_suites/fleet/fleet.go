// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.elastic.co/apm"

	"github.com/cenkalti/backoff/v4"
	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/elasticsearch"
	"github.com/elastic/e2e-testing/internal/installer"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const actionADDED = "added"
const actionREMOVED = "removed"

var deployedAgentsCount = 0

// FleetTestSuite represents the scenarios for Fleet-mode
type FleetTestSuite struct {
	// integrations
	StandAlone          bool
	CurrentToken        string // current enrollment token
	CurrentTokenID      string // current enrollment tokenID
	ElasticAgentStopped bool   // will be used to signal when the agent process can be called again in the tear-down stage
	Image               string // base image used to install the agent
	InstallerType       string
	Integration         kibana.IntegrationPackage // the installed integration
	Policy              kibana.Policy
	PolicyUpdatedAt     string // the moment the policy was updated
	FleetServerPolicy   kibana.Policy
	Version             string // current elastic-agent version
	kibanaClient        *kibana.Client
	deployer            deploy.Deployment
	BeatsProcess        string // (optional) name of the Beats that must be present before installing the elastic-agent
	// date controls for queries
	AgentStoppedDate             time.Time
	RuntimeDependenciesStartDate time.Time
	// instrumentation
	currentContext context.Context
}

// afterScenario destroys the state created by a scenario
func (fts *FleetTestSuite) afterScenario() {
	defer func() { deployedAgentsCount = 0 }()

	span := tx.StartSpan("Clean up", "test.scenario.clean", nil)
	fts.currentContext = apm.ContextWithSpan(context.Background(), span)
	defer span.End()

	serviceName := common.ElasticAgentServiceName
	agentService := deploy.NewServiceRequest(serviceName)

	if !fts.StandAlone {
		agentInstaller, _ := installer.Attach(fts.currentContext, fts.deployer, agentService, fts.InstallerType)

		if log.IsLevelEnabled(log.DebugLevel) {
			err := agentInstaller.Logs()
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
		_ = fts.deployer.Logs(agentService)
	}

	err := fts.unenrollHostname()
	if err != nil {
		manifest, _ := fts.deployer.Inspect(fts.currentContext, agentService)
		log.WithFields(log.Fields{
			"err":      err,
			"hostname": manifest.Hostname,
		}).Warn("The agentIDs for the hostname could not be unenrolled")
	}

	if !common.DeveloperMode {
		_ = fts.deployer.Remove(
			common.FleetProfileServiceRequest,
			[]deploy.ServiceRequest{
				deploy.NewServiceRequest(serviceName),
			},
			common.ProfileEnv)
	} else {
		log.WithField("service", serviceName).Info("Because we are running in development mode, the service won't be stopped")
	}

	err = fts.kibanaClient.DeleteEnrollmentAPIKey(fts.currentContext, fts.CurrentTokenID)
	if err != nil {
		log.WithFields(log.Fields{
			"err":     err,
			"tokenID": fts.CurrentTokenID,
		}).Warn("The enrollment token could not be deleted")
	}

	fts.kibanaClient.DeleteAllPolicies(fts.currentContext)

	// clean up fields
	fts.CurrentTokenID = ""
	fts.CurrentToken = ""
	fts.Image = ""
	fts.StandAlone = false
	fts.BeatsProcess = ""
}

// beforeScenario creates the state needed by a scenario
func (fts *FleetTestSuite) beforeScenario() {
	fts.StandAlone = false
	fts.ElasticAgentStopped = false

	fts.Version = common.BeatVersion

	policy, err := fts.kibanaClient.GetDefaultPolicy(fts.currentContext, false)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warn("The default policy could not be obtained")

	}
	fts.Policy = policy
}

func (fts *FleetTestSuite) contributeSteps(s *godog.ScenarioContext) {
	s.Step(`^a "([^"]*)" agent is deployed to Fleet$`, fts.anAgentIsDeployedToFleet)
	s.Step(`^a "([^"]*)" agent is deployed to Fleet with "([^"]*)" installer$`, fts.anAgentIsDeployedToFleetWithInstaller)
	s.Step(`^a "([^"]*)" agent "([^"]*)" is deployed to Fleet with "([^"]*)" installer$`, fts.anStaleAgentIsDeployedToFleetWithInstaller)
	s.Step(`^agent is in version "([^"]*)"$`, fts.agentInVersion)
	s.Step(`^agent is upgraded to version "([^"]*)"$`, fts.anAgentIsUpgraded)
	s.Step(`^the agent is listed in Fleet as "([^"]*)"$`, fts.theAgentIsListedInFleetWithStatus)
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

	// endpoint steps
	s.Step(`^the "([^"]*)" integration is "([^"]*)" in the policy$`, fts.theIntegrationIsOperatedInThePolicy)
	s.Step(`^the "([^"]*)" datasource is shown in the policy as added$`, fts.thePolicyShowsTheDatasourceAdded)
	s.Step(`^the host name is shown in the Administration view in the Security App as "([^"]*)"$`, fts.theHostNameIsShownInTheAdminViewInTheSecurityApp)
	s.Step(`^the host name is not shown in the Administration view in the Security App$`, fts.theHostNameIsNotShownInTheAdminViewInTheSecurityApp)
	s.Step(`^an "([^"]*)" is successfully deployed with a "([^"]*)" Agent using "([^"]*)" installer$`, fts.anIntegrationIsSuccessfullyDeployedWithAgentAndInstaller)
	s.Step(`^the policy response will be shown in the Security App$`, fts.thePolicyResponseWillBeShownInTheSecurityApp)
	s.Step(`^the policy is updated to have "([^"]*)" in "([^"]*)" mode$`, fts.thePolicyIsUpdatedToHaveMode)
	s.Step(`^the policy will reflect the change in the Security App$`, fts.thePolicyWillReflectTheChangeInTheSecurityApp)

	// stand-alone only steps
	s.Step(`^a "([^"]*)" stand-alone agent is deployed$`, fts.aStandaloneAgentIsDeployed)
	s.Step(`^a "([^"]*)" stand-alone agent is deployed with fleet server mode$`, fts.bootstrapFleetServerFromAStandaloneAgent)
	s.Step(`^a "([^"]*)" stand-alone agent is deployed with fleet server mode on cloud$`, fts.aStandaloneAgentIsDeployedWithFleetServerModeOnCloud)
	s.Step(`^there is new data in the index from agent$`, fts.thereIsNewDataInTheIndexFromAgent)
	s.Step(`^the "([^"]*)" docker container is stopped$`, fts.theDockerContainerIsStopped)
	s.Step(`^there is no new data in the index after agent shuts down$`, fts.thereIsNoNewDataInTheIndexAfterAgentShutsDown)
	s.Step(`^the stand-alone agent is listed in Fleet as "([^"]*)"$`, fts.theStandaloneAgentIsListedInFleetWithStatus)
}

func (fts *FleetTestSuite) theStandaloneAgentIsListedInFleetWithStatus(desiredStatus string) error {
	waitForAgents := func() error {
		agents, err := fts.kibanaClient.ListAgents(fts.currentContext)
		if err != nil {
			return err
		}

		if len(agents) == 0 {
			return errors.New("No agents found")
		}

		agentZero := agents[0]
		hostname := agentZero.LocalMetadata.Host.HostName

		return theAgentIsListedInFleetWithStatus(fts.currentContext, desiredStatus, hostname)
	}
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	exp := utils.GetExponentialBackOff(maxTimeout)

	err := backoff.Retry(waitForAgents, exp)
	if err != nil {
		return err
	}
	return nil
}

func (fts *FleetTestSuite) anStaleAgentIsDeployedToFleetWithInstaller(image, version, installerType string) error {
	agentVersionBackup := fts.Version
	defer func() { fts.Version = agentVersionBackup }()

	common.AgentStaleVersion = shell.GetEnv("ELASTIC_AGENT_STALE_VERSION", common.AgentStaleVersion)
	// check if stale version is an alias
	v, err := utils.GetElasticArtifactVersion(common.AgentStaleVersion)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": common.AgentStaleVersion,
		}).Error("Failed to get stale version")
		return err
	}
	common.AgentStaleVersion = v

	useCISnapshots := shell.GetEnvBool("BEATS_USE_CI_SNAPSHOTS")
	if useCISnapshots && !strings.HasSuffix(common.AgentStaleVersion, "-SNAPSHOT") {
		common.AgentStaleVersion += "-SNAPSHOT"
	}

	switch version {
	case "stale":
		version = common.AgentStaleVersion
	case "latest":
		version = common.BeatVersion
	default:
		version = common.AgentStaleVersion
	}

	fts.Version = version

	return fts.anAgentIsDeployedToFleetWithInstaller(image, installerType)
}

func (fts *FleetTestSuite) installCerts() error {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	agentInstaller, _ := installer.Attach(fts.currentContext, fts.deployer, agentService, fts.InstallerType)

	err := agentInstaller.InstallCerts(fts.currentContext)
	if err != nil {
		log.WithFields(log.Fields{
			"agentVersion":      common.BeatVersion,
			"agentStaleVersion": common.AgentStaleVersion,
			"error":             err,
			"installer":         agentInstaller,
			"version":           fts.Version,
		}).Error("Could not install the certificates")
		return err
	}

	return nil
}

func (fts *FleetTestSuite) anAgentIsUpgraded(desiredVersion string) error {
	switch desiredVersion {
	case "stale":
		desiredVersion = common.AgentStaleVersion
	case "latest":
		desiredVersion = common.BeatVersion
	default:
		desiredVersion = common.BeatVersion
	}

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.deployer.Inspect(fts.currentContext, agentService)
	return fts.kibanaClient.UpgradeAgent(fts.currentContext, manifest.Hostname, desiredVersion)
}

func (fts *FleetTestSuite) agentInVersion(version string) error {
	switch version {
	case "stale":
		version = common.AgentStaleVersion
	case "latest":
		version = common.BeatVersion
	}

	agentInVersionFn := func() error {
		agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
		manifest, _ := fts.deployer.Inspect(fts.currentContext, agentService)
		agent, err := fts.kibanaClient.GetAgentByHostname(fts.currentContext, manifest.Hostname)
		if err != nil {
			return err
		}

		retrievedVersion := agent.LocalMetadata.Elastic.Agent.Version
		if isSnapshot := agent.LocalMetadata.Elastic.Agent.Snapshot; isSnapshot {
			retrievedVersion += "-SNAPSHOT"
		}

		if retrievedVersion != version {
			return fmt.Errorf("version mismatch required '%s' retrieved '%s'", version, retrievedVersion)
		}

		return nil
	}

	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	exp := utils.GetExponentialBackOff(maxTimeout)

	return backoff.Retry(agentInVersionFn, exp)
}

// this step infers the installer type from the underlying OS image
// supported images: centos and debian
func (fts *FleetTestSuite) anAgentIsDeployedToFleet(image string) error {
	installerType := "rpm"
	if image == "debian" {
		installerType = "deb"
	}

	return fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(image, installerType)
}

// supported installers: tar, rpm, deb
func (fts *FleetTestSuite) anAgentIsDeployedToFleetWithInstaller(image string, installerType string) error {
	fts.BeatsProcess = ""
	return fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(image, installerType)
}

func (fts *FleetTestSuite) anAgentIsDeployedToFleetWithInstallerAndFleetServer(image string, installerType string) error {
	log.WithFields(log.Fields{
		"image":     image,
		"installer": installerType,
	}).Trace("Deploying an agent to Fleet with base image using an already bootstrapped Fleet Server")

	deployedAgentsCount++

	fts.Image = image
	fts.InstallerType = installerType

	// Grab a new enrollment key for new agent
	enrollmentKey, err := fts.kibanaClient.CreateEnrollmentAPIKey(fts.currentContext, fts.Policy)
	if err != nil {
		return err
	}
	fts.CurrentToken = enrollmentKey.APIKey
	fts.CurrentTokenID = enrollmentKey.ID

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName).WithFlavour(image).WithScale(deployedAgentsCount)
	if fts.BeatsProcess != "" {
		agentService = agentService.WithBackgroundProcess(fts.BeatsProcess)
	}

	services := []deploy.ServiceRequest{
		agentService,
	}

	err = fts.deployer.Add(fts.currentContext, common.FleetProfileServiceRequest, services, common.ProfileEnv)
	if err != nil {
		return err
	}

	agentInstaller, _ := installer.Attach(fts.currentContext, fts.deployer, agentService, installerType)
	err = deployAgentToFleet(fts.currentContext, agentInstaller, fts.CurrentToken)
	if err != nil {
		return err
	}
	return err
}

func (fts *FleetTestSuite) processStateChangedOnTheHost(process string, state string) error {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	agentInstaller, _ := installer.Attach(fts.currentContext, fts.deployer, agentService, fts.InstallerType)
	if state == "started" {
		err := agentInstaller.Start(fts.currentContext)
		return err
	} else if state == "restarted" {
		err := agentInstaller.Stop(fts.currentContext)
		if err != nil {
			return err
		}

		utils.Sleep(time.Duration(utils.TimeoutFactor) * 10 * time.Second)

		err = agentInstaller.Start(fts.currentContext)
		if err != nil {
			return err
		}
		return nil
	} else if state == "uninstalled" {
		err := agentInstaller.Uninstall(fts.currentContext)
		if err != nil {
			return err
		}

		// signal that the elastic-agent was uninstalled
		if process == common.ElasticAgentProcessName {
			fts.ElasticAgentStopped = true
		}

		return nil
	} else if state != "stopped" {
		return godog.ErrPending
	}

	log.WithFields(log.Fields{
		"service": agentService.Name,
		"process": process,
	}).Trace("Stopping process on the service")

	err := agentInstaller.Stop(fts.currentContext)
	if err != nil {
		log.WithFields(log.Fields{
			"action":  state,
			"error":   err,
			"service": agentService.Name,
			"process": process,
		}).Error("Could not stop process on the host")

		return err
	}

	manifest, _ := fts.deployer.Inspect(fts.currentContext, agentService)

	return CheckProcessState(fts.deployer, manifest.Name, process, "stopped", 0, utils.TimeoutFactor)
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
	manifest, _ := fts.deployer.Inspect(fts.currentContext, agentService)
	return theAgentIsListedInFleetWithStatus(fts.currentContext, desiredStatus, manifest.Hostname)
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
			return fmt.Errorf("The agent is not present in Fleet in the '%s' status, but it should", desiredStatus)
		}

		agentStatus, err := kibanaClient.GetAgentStatusByHostname(ctx, hostname)
		isAgentInStatus := strings.EqualFold(agentStatus, desiredStatus)
		if err != nil || !isAgentInStatus {
			if err == nil {
				err = fmt.Errorf("The Agent is not in the %s status yet", desiredStatus)
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
	agentInstaller, _ := installer.Attach(fts.currentContext, fts.deployer, agentService, fts.InstallerType)

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

	return fmt.Errorf("The file system directory is not empty")
}

func (fts *FleetTestSuite) theHostIsRestarted() error {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	err := fts.deployer.Stop(agentService)
	if err != nil {
		log.WithField("err", err).Error("Could not stop the service")
	}

	utils.Sleep(time.Duration(utils.TimeoutFactor) * 10 * time.Second)

	err = fts.deployer.Start(agentService)
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
			err = fmt.Errorf("There are no datastreams yet")

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
		err = fmt.Errorf("There are no datastreams. We expected to have more than one")
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
	agentInstaller, _ := installer.Attach(fts.currentContext, fts.deployer, agentService, fts.InstallerType)

	err := agentInstaller.Enroll(fts.currentContext, fts.CurrentToken)
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

	return nil
}

func (fts *FleetTestSuite) theIntegrationIsOperatedInThePolicy(packageName string, action string) error {
	return theIntegrationIsOperatedInThePolicy(fts.currentContext, fts.kibanaClient, fts.Policy, packageName, action)
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
			Name:        integration.Name,
			Description: integration.Title,
			Namespace:   "default",
			PolicyID:    policy.ID,
			Enabled:     true,
			Package:     integration,
			Inputs:      []kibana.Input{},
		}
		packageDataStream.Inputs = inputs(integration.Name)

		return client.AddIntegrationToPolicy(ctx, packageDataStream)
	} else if strings.ToLower(action) == actionREMOVED {
		packageDataStream, err := client.GetIntegrationFromAgentPolicy(ctx, integration.Name, policy)
		if err != nil {
			return err
		}
		return client.DeleteIntegrationFromPolicy(ctx, packageDataStream)
	}

	return nil
}

func (fts *FleetTestSuite) theHostNameIsNotShownInTheAdminViewInTheSecurityApp() error {
	log.Trace("Checking if the hostname is not shown in the Administration view in the Security App")

	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	agentListedInSecurityFn := func() error {
		agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
		manifest, _ := fts.deployer.Inspect(fts.currentContext, agentService)
		host, err := fts.kibanaClient.IsAgentListedInSecurityApp(fts.currentContext, manifest.Hostname)
		if err != nil {
			log.WithFields(log.Fields{
				"elapsedTime": exp.GetElapsedTime(),
				"err":         err,
				"host":        host,
				"hostname":    manifest.Hostname,
				"retry":       retryCount,
			}).Warn("We could not check the agent in the Administration view in the Security App yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"elapsedTime": exp.GetElapsedTime(),
			"hostname":    manifest.Hostname,
			"retries":     retryCount,
		}).Info("The Agent is not listed in the Administration view in the Security App")
		return nil
	}

	err := backoff.Retry(agentListedInSecurityFn, exp)
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) theHostNameIsShownInTheAdminViewInTheSecurityApp(status string) error {
	log.Trace("Checking if the hostname is shown in the Admin view in the Security App")

	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	agentListedInSecurityFn := func() error {
		agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
		manifest, _ := fts.deployer.Inspect(fts.currentContext, agentService)
		matches, err := fts.kibanaClient.IsAgentListedInSecurityAppWithStatus(fts.currentContext, manifest.Hostname, status)
		if err != nil || !matches {
			log.WithFields(log.Fields{
				"elapsedTime":   exp.GetElapsedTime(),
				"desiredStatus": status,
				"err":           err,
				"hostname":      manifest.Hostname,
				"matches":       matches,
				"retry":         retryCount,
			}).Warn("The agent is not listed in the Administration view in the Security App in the desired status yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"elapsedTime":   exp.GetElapsedTime(),
			"desiredStatus": status,
			"hostname":      manifest.Hostname,
			"matches":       matches,
			"retries":       retryCount,
		}).Info("The Agent is listed in the Administration view in the Security App in the desired status")
		return nil
	}

	err := backoff.Retry(agentListedInSecurityFn, exp)
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) anIntegrationIsSuccessfullyDeployedWithAgentAndInstaller(integration string, image string, installerType string) error {
	err := fts.anAgentIsDeployedToFleetWithInstaller(image, installerType)
	if err != nil {
		return err
	}

	return fts.theIntegrationIsOperatedInThePolicy(integration, actionADDED)
}

func (fts *FleetTestSuite) thePolicyResponseWillBeShownInTheSecurityApp() error {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.deployer.Inspect(fts.currentContext, agentService)
	agentID, err := fts.kibanaClient.GetAgentIDByHostname(fts.currentContext, manifest.Hostname)
	if err != nil {
		return err
	}

	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	getEventsFn := func() error {
		listed, err := fts.kibanaClient.IsPolicyResponseListedInSecurityApp(fts.currentContext, agentID)
		if err != nil {
			log.WithFields(log.Fields{
				"elapsedTime": exp.GetElapsedTime(),
				"err":         err,
				"retries":     retryCount,
			}).Warn("Could not get metadata from the Administration view in the Security App yet")
			retryCount++

			return err
		}

		if !listed {
			log.WithFields(log.Fields{
				"agentID":     agentID,
				"elapsedTime": exp.GetElapsedTime(),
				"retries":     retryCount,
			}).Warn("The policy response is not listed as 'success' in the Administration view in the Security App yet")
			retryCount++

			return fmt.Errorf("The policy response is not listed as 'success' in the Administration view in the Security App yet")
		}

		log.WithFields(log.Fields{
			"elapsedTime": exp.GetElapsedTime(),
			"retries":     retryCount,
		}).Info("The policy response is listed as 'success' in the Administration view in the Security App")
		return nil
	}

	err = backoff.Retry(getEventsFn, exp)
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) thePolicyIsUpdatedToHaveMode(name string, mode string) error {
	if name != "malware" {
		log.WithFields(log.Fields{
			"name": name,
		}).Warn("We only support 'malware' policy to be updated")
		return godog.ErrPending
	}

	if mode != "detect" && mode != "prevent" {
		log.WithFields(log.Fields{
			"name": name,
			"mode": mode,
		}).Warn("We only support 'detect' and 'prevent' modes")
		return godog.ErrPending
	}

	packageDS, err := fts.kibanaClient.GetIntegrationFromAgentPolicy(fts.currentContext, "endpoint", fts.Policy)

	if err != nil {
		return err
	}
	fts.Integration = packageDS.Package

	for _, item := range packageDS.Inputs {
		if item.Type == "endpoint" {
			item.Config.(map[string]interface{})["policy"].(map[string]interface{})["value"].(map[string]interface{})["windows"].(map[string]interface{})["malware"].(map[string]interface{})["mode"] = mode
			item.Config.(map[string]interface{})["policy"].(map[string]interface{})["value"].(map[string]interface{})["mac"].(map[string]interface{})["malware"].(map[string]interface{})["mode"] = mode
		}
	}
	log.WithFields(log.Fields{
		"inputs": packageDS.Inputs,
	}).Trace("Upgrading integration package config")

	updatedAt, err := fts.kibanaClient.UpdateIntegrationPackagePolicy(fts.currentContext, packageDS)
	if err != nil {
		return err
	}

	// we use a string because we are not able to process what comes in the event, so we will do
	// an alphabetical order, as they share same layout but different millis and timezone format
	fts.PolicyUpdatedAt = updatedAt
	return nil
}

func (fts *FleetTestSuite) thePolicyWillReflectTheChangeInTheSecurityApp() error {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.deployer.Inspect(fts.currentContext, agentService)
	agentID, err := fts.kibanaClient.GetAgentIDByHostname(fts.currentContext, manifest.Hostname)
	if err != nil {
		return err
	}

	pkgPolicy, err := fts.kibanaClient.GetIntegrationFromAgentPolicy(fts.currentContext, "endpoint", fts.Policy)
	if err != nil {
		return err
	}

	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	getEventsFn := func() error {
		err := fts.kibanaClient.GetAgentEvents(fts.currentContext, "endpoint-security", agentID, pkgPolicy.ID, fts.PolicyUpdatedAt)
		if err != nil {
			log.WithFields(log.Fields{
				"elapsedTime": exp.GetElapsedTime(),
				"err":         err,
				"retries":     retryCount,
			}).Warn("There are no events for the agent in Fleet")
			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"elapsedTime": exp.GetElapsedTime(),
			"retries":     retryCount,
		}).Info("There are events for the agent in Fleet")
		return nil
	}

	err = backoff.Retry(getEventsFn, exp)
	if err != nil {
		return err
	}

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

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName).WithFlavour(fts.Image).WithScale(deployedAgentsCount)
	services := []deploy.ServiceRequest{
		agentService,
	}

	err := fts.deployer.Add(fts.currentContext, common.FleetProfileServiceRequest, services, common.ProfileEnv)
	if err != nil {
		return err
	}

	agentInstaller, _ := installer.Attach(fts.currentContext, fts.deployer, agentService, fts.InstallerType)
	err = deployAgentToFleet(fts.currentContext, agentInstaller, fts.CurrentToken)

	if err == nil {
		err = fmt.Errorf("The agent was enrolled although the token was previously revoked")

		log.WithFields(log.Fields{
			"tokenID": fts.CurrentTokenID,
			"error":   err,
		}).Error(err.Error())
		return err
	}

	// checking the error message produced by the install command in TAR installer
	// to distinguish from other install errors
	if err != nil && strings.HasPrefix(err.Error(), "Failed to install the agent with subcommand:") {
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
	manifest, _ := fts.deployer.Inspect(fts.currentContext, agentService)
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

	_, err := elasticsearch.WaitForNumberOfHits(context.Background(), indexName, query, 1, time.Minute)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn(elasticsearch.WaitForIndices())
	}

	return err
}

func deployAgentToFleet(ctx context.Context, agentInstaller deploy.ServiceOperator, token string) error {
	err := agentInstaller.Preinstall(ctx)
	if err != nil {
		return err
	}

	err = agentInstaller.Install(ctx)
	if err != nil {
		return err
	}

	err = agentInstaller.Enroll(ctx, token)
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
				Streams: []interface{}{},
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
				Streams: []interface{}{
					map[string]interface{}{
						"id":      "linux/metrics-linux.memory-" + uuid.New().String(),
						"enabled": true,
						"data_stream": map[string]interface{}{
							"dataset": "linux.memory",
							"type":    "metrics",
						},
					},
				},
				Vars: map[string]kibana.Var{
					"period": {
						Value: "1s",
						Type:  "string",
					},
				},
			},
		}
	}
	return []kibana.Input{}
}
