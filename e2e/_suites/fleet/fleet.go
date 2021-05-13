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

// FleetTestSuite represents the scenarios for Fleet-mode
type FleetTestSuite struct {
	// integrations
	StandAlone          bool
	CurrentToken        string // current enrollment token
	CurrentTokenID      string // current enrollment tokenID
	ElasticAgentStopped bool   // will be used to signal when the agent process can be called again in the tear-down stage
	Hostname            string // the hostname of the container
	Image               string // base image used to install the agent
	InstallerType       string
	Installers          map[string]installer.ElasticAgentInstaller
	Integration         kibana.IntegrationPackage // the installed integration
	Policy              kibana.Policy
	PolicyUpdatedAt     string // the moment the policy was updated
	FleetServerPolicy   kibana.Policy
	Version             string // current elastic-agent version
	kibanaClient        *kibana.Client
	deployer            deploy.Deployment
	// date controls for queries
	AgentStoppedDate             time.Time
	RuntimeDependenciesStartDate time.Time
}

// afterScenario destroys the state created by a scenario
func (fts *FleetTestSuite) afterScenario() {
	serviceName := common.ElasticAgentServiceName

	if !fts.StandAlone {
		agentInstaller := fts.getInstaller()

		if log.IsLevelEnabled(log.DebugLevel) {
			err := agentInstaller.PrintLogsFn(fts.Hostname)
			if err != nil {
				log.WithFields(log.Fields{
					"containerName": fts.Hostname,
					"error":         err,
				}).Warn("Could not get agent logs in the container")
			}
		}
		// only call it when the elastic-agent is present
		if !fts.ElasticAgentStopped {
			err := agentInstaller.UninstallFn()
			if err != nil {
				log.Warnf("Could not uninstall the agent after the scenario: %v", err)
			}
		}
	} else if log.IsLevelEnabled(log.DebugLevel) {
		_ = fts.getContainerLogs()
	}

	err := fts.unenrollHostname()
	if err != nil {
		log.WithFields(log.Fields{
			"err":      err,
			"hostname": fts.Hostname,
		}).Warn("The agentIDs for the hostname could not be unenrolled")
	}

	developerMode := shell.GetEnvBool("DEVELOPER_MODE")
	if !developerMode {
		image := ""
		if !fts.StandAlone {
			agentInstaller := fts.getInstaller()
			image = agentInstaller.Image
		}

		_ = fts.deployer.Remove(
			[]deploy.ServiceRequest{
				deploy.NewServiceRequest(common.FleetProfileName),
				deploy.NewServiceRequest(serviceName).WithFlavour(image),
			},
			common.ProfileEnv)
	} else {
		log.WithField("service", serviceName).Info("Because we are running in development mode, the service won't be stopped")
	}

	err = fts.kibanaClient.DeleteEnrollmentAPIKey(fts.CurrentTokenID)
	if err != nil {
		log.WithFields(log.Fields{
			"err":     err,
			"tokenID": fts.CurrentTokenID,
		}).Warn("The enrollment token could not be deleted")
	}

	fts.kibanaClient.DeleteAllPolicies()

	// clean up fields
	fts.CurrentTokenID = ""
	fts.CurrentToken = ""
	fts.Image = ""
	fts.Hostname = ""
	fts.StandAlone = false
}

// beforeScenario creates the state needed by a scenario
func (fts *FleetTestSuite) beforeScenario() {
	fts.StandAlone = false
	fts.ElasticAgentStopped = false

	fts.Version = common.BeatVersion

	policy, err := fts.kibanaClient.GetDefaultPolicy(false)
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
		agents, err := fts.kibanaClient.ListAgents()
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

	// prepare installer for stale version
	if fts.Version != agentVersionBackup {
		i := installer.GetElasticAgentInstaller(image, installerType, fts.Version)
		fts.Installers[fmt.Sprintf("%s-%s-%s", image, installerType, version)] = i
	}

	return fts.anAgentIsDeployedToFleetWithInstaller(image, installerType)
}

func (fts *FleetTestSuite) installCerts() error {
	agentInstaller := fts.getInstaller()
	if agentInstaller.InstallCertsFn == nil {
		log.WithFields(log.Fields{
			"installer":         agentInstaller,
			"version":           fts.Version,
			"agentVersion":      common.BeatVersion,
			"agentStaleVersion": common.AgentStaleVersion,
		}).Error("No installer found")
		return errors.New("no installer found")
	}

	err := agentInstaller.InstallCertsFn()
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

	return fts.kibanaClient.UpgradeAgent(fts.Hostname, desiredVersion)
}

func (fts *FleetTestSuite) agentInVersion(version string) error {
	switch version {
	case "stale":
		version = common.AgentStaleVersion
	case "latest":
		version = common.BeatVersion
	}

	agentInVersionFn := func() error {
		agent, err := fts.kibanaClient.GetAgentByHostname(fts.Hostname)
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
	return fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(image, installerType)
}

func (fts *FleetTestSuite) anAgentIsDeployedToFleetWithInstallerAndFleetServer(image string, installerType string) error {
	log.WithFields(log.Fields{
		"image":     image,
		"installer": installerType,
	}).Trace("Deploying an agent to Fleet with base image using an already bootstrapped Fleet Server")

	fts.Image = image
	fts.InstallerType = installerType

	agentInstaller := fts.getInstaller()

	containerName := fts.getContainerName(agentInstaller, 1) // name of the container

	// enroll the agent with a new token
	enrollmentKey, err := fts.kibanaClient.CreateEnrollmentAPIKey(fts.Policy)
	if err != nil {
		return err
	}
	fts.CurrentToken = enrollmentKey.APIKey
	fts.CurrentTokenID = enrollmentKey.ID

	var fleetConfig *kibana.FleetConfig
	fleetConfig, err = deployAgentToFleet(agentInstaller, fts.deployer, containerName, fts.CurrentToken)

	if err != nil {
		return err
	}

	// the installation process for TAR includes the enrollment
	if agentInstaller.InstallerType != "tar" {
		err = agentInstaller.EnrollFn(fleetConfig)
		if err != nil {
			return err
		}
	}

	// get container hostname once
	hostname, err := deploy.GetContainerHostname(containerName)
	if err != nil {
		return err
	}
	fts.Hostname = hostname

	return err
}

// getContainerName returns the current container name for the service:
// we are using the Docker client instead of docker-compose because it does not support
// returning the output of a command: it simply returns error level
func (fts *FleetTestSuite) getContainerName(i installer.ElasticAgentInstaller, index int) string {
	return fmt.Sprintf("%s_%s_%d", i.Profile, common.ElasticAgentServiceName, index)
}

// getServiceName returns the current service name, the one defined at the docker compose
func (fts *FleetTestSuite) getServiceName(i installer.ElasticAgentInstaller) string {
	return i.Image
}

func (fts *FleetTestSuite) getInstaller() installer.ElasticAgentInstaller {
	key := fmt.Sprintf("%s-%s-%s", fts.Image, fts.InstallerType, fts.Version)
	// check if the agent is already cached
	if i, exists := fts.Installers[key]; exists {
		return i
	}

	agentInstaller := installer.GetElasticAgentInstaller(fts.Image, fts.InstallerType, fts.Version)

	// cache the new installer
	fts.Installers[key] = agentInstaller

	return agentInstaller
}

func (fts *FleetTestSuite) processStateChangedOnTheHost(process string, state string) error {
	profile := common.FleetProfileName

	agentInstaller := fts.getInstaller()

	serviceName := agentInstaller.Service // name of the service

	profileService := deploy.NewServiceRequest(profile)
	imageService := deploy.NewServiceRequest(common.ElasticAgentServiceName).WithFlavour(agentInstaller.Image)

	if state == "started" {
		return installer.SystemctlRun(profileService, imageService, serviceName, "start")
	} else if state == "restarted" {
		err := installer.SystemctlRun(profileService, imageService, serviceName, "stop")
		if err != nil {
			return err
		}

		utils.Sleep(time.Duration(utils.TimeoutFactor) * 10 * time.Second)

		err = installer.SystemctlRun(profileService, imageService, serviceName, "start")
		if err != nil {
			return err
		}
		return nil
	} else if state == "uninstalled" {
		err := agentInstaller.UninstallFn()
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
		"service": serviceName,
		"process": process,
	}).Trace("Stopping process on the service")

	err := installer.SystemctlRun(profileService, imageService, serviceName, "stop")
	if err != nil {
		log.WithFields(log.Fields{
			"action":  state,
			"error":   err,
			"service": serviceName,
			"process": process,
		}).Error("Could not stop process on the host")

		return err
	}

	containerName := fts.getContainerName(agentInstaller, 1)

	return CheckProcessState(fts.deployer, containerName, process, "stopped", 1, utils.TimeoutFactor)
}

func (fts *FleetTestSuite) setup() error {
	log.Trace("Creating Fleet setup")

	err := fts.kibanaClient.RecreateFleet()
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) theAgentIsListedInFleetWithStatus(desiredStatus string) error {
	return theAgentIsListedInFleetWithStatus(desiredStatus, fts.Hostname)
}

func theAgentIsListedInFleetWithStatus(desiredStatus string, hostname string) error {
	log.Tracef("Checking if agent is listed in Fleet as %s", desiredStatus)

	kibanaClient, err := kibana.NewClient()
	if err != nil {
		return err
	}
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	agentOnlineFn := func() error {
		agentID, err := kibanaClient.GetAgentIDByHostname(hostname)
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

		agentStatus, err := kibanaClient.GetAgentStatusByHostname(hostname)
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
	agentInstaller := fts.getInstaller()

	containerName := fts.getContainerName(agentInstaller, 1)

	content, err := agentInstaller.ListElasticAgentWorkingDirContent(containerName)
	if err != nil {
		return err
	}

	if content == "" || strings.Contains(content, "No such file or directory") {
		return nil
	}

	return fmt.Errorf("The file system directory is not empty")
}

func (fts *FleetTestSuite) theHostIsRestarted() error {
	agentInstaller := fts.getInstaller()

	containerName := fts.getContainerName(agentInstaller, 1)
	_, err := shell.Execute(context.Background(), ".", "docker", "stop", containerName)
	if err != nil {
		log.WithFields(log.Fields{
			"containerName": containerName,
			"image":         agentInstaller.Image,
			"service":       agentInstaller.Service,
		}).Error("Could not stop the service")
	}

	utils.Sleep(time.Duration(utils.TimeoutFactor) * 10 * time.Second)

	_, err = shell.Execute(context.Background(), ".", "docker", "start", containerName)
	if err != nil {
		log.WithFields(log.Fields{
			"containerName": containerName,
			"image":         agentInstaller.Image,
			"service":       agentInstaller.Service,
		}).Error("Could not start the service")
	}

	log.WithFields(log.Fields{
		"containerName": containerName,
		"image":         agentInstaller.Image,
		"service":       agentInstaller.Service,
	}).Debug("The service has been restarted")
	return nil
}

func (fts *FleetTestSuite) systemPackageDashboardsAreListedInFleet() error {
	log.Trace("Checking system Package dashboards in Fleet")

	dataStreamsCount := 0
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	countDataStreamsFn := func() error {
		dataStreams, err := fts.kibanaClient.GetDataStreams()
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

	agentInstaller := fts.getInstaller()

	// a re-enroll does need to bootstrap the Fleet Server again
	// during an unenroll the fleet server exits as there is no longer
	// and agent id associated with the enrollment. When fleet server
	// restarts it needs a new agent to associate with the boostrap
	cfg, err := kibana.NewFleetConfig(fts.CurrentToken)
	if err != nil {
		return err
	}

	err = agentInstaller.EnrollFn(cfg)
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

	err := fts.kibanaClient.DeleteEnrollmentAPIKey(fts.CurrentTokenID)
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
	return theIntegrationIsOperatedInThePolicy(fts.kibanaClient, fts.Policy, packageName, action)
}

func theIntegrationIsOperatedInThePolicy(client *kibana.Client, policy kibana.Policy, packageName string, action string) error {
	log.WithFields(log.Fields{
		"action":  action,
		"policy":  policy,
		"package": packageName,
	}).Trace("Doing an operation for a package on a policy")

	integration, err := client.GetIntegrationByPackageName(packageName)
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

		return client.AddIntegrationToPolicy(packageDataStream)
	} else if strings.ToLower(action) == actionREMOVED {
		packageDataStream, err := client.GetIntegrationFromAgentPolicy(integration.Name, policy)
		if err != nil {
			return err
		}
		return client.DeleteIntegrationFromPolicy(packageDataStream)
	}

	return nil
}

func (fts *FleetTestSuite) theHostNameIsNotShownInTheAdminViewInTheSecurityApp() error {
	log.Trace("Checking if the hostname is not shown in the Administration view in the Security App")

	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	agentListedInSecurityFn := func() error {
		host, err := fts.kibanaClient.IsAgentListedInSecurityApp(fts.Hostname)
		if err != nil {
			log.WithFields(log.Fields{
				"elapsedTime": exp.GetElapsedTime(),
				"err":         err,
				"host":        host,
				"hostname":    fts.Hostname,
				"retry":       retryCount,
			}).Warn("We could not check the agent in the Administration view in the Security App yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"elapsedTime": exp.GetElapsedTime(),
			"hostname":    fts.Hostname,
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
		matches, err := fts.kibanaClient.IsAgentListedInSecurityAppWithStatus(fts.Hostname, status)
		if err != nil || !matches {
			log.WithFields(log.Fields{
				"elapsedTime":   exp.GetElapsedTime(),
				"desiredStatus": status,
				"err":           err,
				"hostname":      fts.Hostname,
				"matches":       matches,
				"retry":         retryCount,
			}).Warn("The agent is not listed in the Administration view in the Security App in the desired status yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"elapsedTime":   exp.GetElapsedTime(),
			"desiredStatus": status,
			"hostname":      fts.Hostname,
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
	agentID, err := fts.kibanaClient.GetAgentIDByHostname(fts.Hostname)
	if err != nil {
		return err
	}

	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	getEventsFn := func() error {
		listed, err := fts.kibanaClient.IsPolicyResponseListedInSecurityApp(agentID)
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

	packageDS, err := fts.kibanaClient.GetIntegrationFromAgentPolicy("endpoint", fts.Policy)

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

	updatedAt, err := fts.kibanaClient.UpdateIntegrationPackagePolicy(packageDS)
	if err != nil {
		return err
	}

	// we use a string because we are not able to process what comes in the event, so we will do
	// an alphabetical order, as they share same layout but different millis and timezone format
	fts.PolicyUpdatedAt = updatedAt
	return nil
}

func (fts *FleetTestSuite) thePolicyWillReflectTheChangeInTheSecurityApp() error {
	agentID, err := fts.kibanaClient.GetAgentIDByHostname(fts.Hostname)
	if err != nil {
		return err
	}

	pkgPolicy, err := fts.kibanaClient.GetIntegrationFromAgentPolicy("endpoint", fts.Policy)
	if err != nil {
		return err
	}

	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	getEventsFn := func() error {
		err := fts.kibanaClient.GetAgentEvents("endpoint-security", agentID, pkgPolicy.ID, fts.PolicyUpdatedAt)
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

	integration, err := fts.kibanaClient.GetIntegrationByPackageName(packageName)
	if err != nil {
		return err
	}

	_, err = fts.kibanaClient.InstallIntegrationAssets(integration)
	if err != nil {
		return err
	}
	fts.Integration = integration

	return nil
}

func (fts *FleetTestSuite) anAttemptToEnrollANewAgentFails() error {
	log.Trace("Enrolling a new agent with an revoked token")

	agentInstaller := fts.getInstaller()

	containerName := fts.getContainerName(agentInstaller, 2) // name of the new container

	fleetConfig, err := deployAgentToFleet(agentInstaller, fts.deployer, containerName, fts.CurrentToken)

	// the installation process for TAR includes the enrollment
	if agentInstaller.InstallerType != "tar" {
		if err != nil {
			return err
		}

		err = agentInstaller.EnrollFn(fleetConfig)
		if err == nil {
			err = fmt.Errorf("The agent was enrolled although the token was previously revoked")

			log.WithFields(log.Fields{
				"tokenID": fts.CurrentTokenID,
				"error":   err,
			}).Error(err.Error())

			return err
		}

		log.WithFields(log.Fields{
			"err":   err,
			"token": fts.CurrentToken,
		}).Debug("As expected, it's not possible to enroll an agent with a revoked token")
		return nil
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

	return err
}

// unenrollHostname deletes the statuses for an existing agent, filtering by hostname
func (fts *FleetTestSuite) unenrollHostname() error {
	log.Tracef("Un-enrolling all agentIDs for %s", fts.Hostname)

	agents, err := fts.kibanaClient.ListAgents()
	if err != nil {
		return err
	}

	for _, agent := range agents {
		if agent.LocalMetadata.Host.HostName == fts.Hostname {
			log.WithFields(log.Fields{
				"hostname": fts.Hostname,
			}).Debug("Un-enrolling agent in Fleet")

			err := fts.kibanaClient.UnEnrollAgent(agent.LocalMetadata.Host.HostName)
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

func deployAgentToFleet(agentInstaller installer.ElasticAgentInstaller, deployer deploy.Deployment, containerName string, token string) (*kibana.FleetConfig, error) {
	profile := agentInstaller.Profile // name of the runtime dependencies compose file
	service := agentInstaller.Service // name of the service
	serviceTag := agentInstaller.Tag  // docker tag of the service

	envVarsPrefix := strings.ReplaceAll(service, "-", "_")

	// let's start with Centos 7
	common.ProfileEnv[envVarsPrefix+"Tag"] = serviceTag
	// we are setting the container name because Centos service could be reused by any other test suite
	common.ProfileEnv[envVarsPrefix+"ContainerName"] = containerName

	services := []deploy.ServiceRequest{
		deploy.NewServiceRequest(profile),
		deploy.NewServiceRequest(common.ElasticAgentServiceName).WithFlavour(agentInstaller.Image),
	}
	err := deployer.Add(services, common.ProfileEnv)
	if err != nil {
		log.WithFields(log.Fields{
			"service": service,
			"tag":     serviceTag,
		}).Error("Could not run the target box")
		return nil, err
	}

	isTar := (agentInstaller.InstallerType == "tar")
	targetFile := "/"

	// copy downloaded agent to the root dir of the container
	err = deploy.CopyFileToContainer(context.Background(), containerName, agentInstaller.BinaryPath, targetFile, isTar)
	if err != nil {
		return nil, err
	}

	err = agentInstaller.PreInstallFn()
	if err != nil {
		return nil, err
	}

	cfg, cfgError := kibana.NewFleetConfig(token)
	if cfgError != nil {
		return nil, cfgError
	}

	err = agentInstaller.InstallFn(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, agentInstaller.PostInstallFn()
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

func (fts *FleetTestSuite) getContainerLogs() error {
	serviceManager := deploy.NewServiceManager()

	image := ""
	if !fts.StandAlone {
		agentInstaller := fts.getInstaller()
		image = agentInstaller.Image
	}

	profile := deploy.NewServiceRequest(common.FleetProfileName)
	serviceName := common.ElasticAgentServiceName

	services := []deploy.ServiceRequest{
		profile, // profile name
		deploy.NewServiceRequest(serviceName).WithFlavour(image), // agent service
	}
	err := serviceManager.RunCommand(profile, services, []string{"logs", serviceName}, common.ProfileEnv)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"service": serviceName,
		}).Error("Could not retrieve Elastic Agent logs")

		return err
	}

	return nil
}
