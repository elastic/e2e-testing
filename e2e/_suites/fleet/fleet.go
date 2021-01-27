// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/cenkalti/backoff/v4"
	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/cli/services"
	curl "github.com/elastic/e2e-testing/cli/shell"
	"github.com/elastic/e2e-testing/e2e"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const fleetAgentsURL = kibanaBaseURL + "/api/fleet/agents"
const fleetAgentEventsURL = kibanaBaseURL + "/api/fleet/agents/%s/events"
const fleetAgentsUnEnrollURL = kibanaBaseURL + "/api/fleet/agents/%s/unenroll"
const fleetAgentUpgradeURL = kibanaBaseURL + "/api/fleet/agents/%s/upgrade"
const fleetEnrollmentTokenURL = kibanaBaseURL + "/api/fleet/enrollment-api-keys"
const fleetSetupURL = kibanaBaseURL + "/api/fleet/agents/setup"
const ingestManagerAgentPoliciesURL = kibanaBaseURL + "/api/fleet/agent_policies"
const ingestManagerDataStreamsURL = kibanaBaseURL + "/api/fleet/data_streams"

const actionADDED = "added"
const actionREMOVED = "removed"

// FleetTestSuite represents the scenarios for Fleet-mode
type FleetTestSuite struct {
	Image          string // base image used to install the agent
	InstallerType  string
	Installers     map[string]ElasticAgentInstaller
	Cleanup        bool
	PolicyID       string // will be used to manage tokens
	CurrentToken   string // current enrollment token
	CurrentTokenID string // current enrollment tokenID
	Hostname       string // the hostname of the container
	Version        string // current elastic-agent version
	// integrations
	Integration     IntegrationPackage // the installed integration
	PolicyUpdatedAt string             // the moment the policy was updated
}

// afterScenario destroys the state created by a scenario
func (fts *FleetTestSuite) afterScenario() {
	serviceManager := services.NewServiceManager()

	serviceName := fts.Image

	if log.IsLevelEnabled(log.DebugLevel) {
		installer := fts.getInstaller()

		if developerMode {
			_ = installer.getElasticAgentLogs(fts.Hostname)
		}

		err := installer.UninstallFn()
		if err != nil {
			log.Warnf("Could not uninstall the agent after the scenario: %v", err)
		}
	}

	err := fts.unenrollHostname(true)
	if err != nil {
		log.WithFields(log.Fields{
			"err":      err,
			"hostname": fts.Hostname,
		}).Warn("The agentIDs for the hostname could not be unenrolled")
	}

	if !developerMode {
		_ = serviceManager.RemoveServicesFromCompose(FleetProfileName, []string{serviceName + "-systemd"}, profileEnv)
	} else {
		log.WithField("service", serviceName).Info("Because we are running in development mode, the service won't be stopped")
	}

	err = fts.removeToken()
	if err != nil {
		log.WithFields(log.Fields{
			"err":     err,
			"tokenID": fts.CurrentTokenID,
		}).Warn("The enrollment token could not be deleted")
	}

	err = deleteIntegrationFromPolicy(fts.Integration, fts.PolicyID)
	if err != nil {
		log.WithFields(log.Fields{
			"err":             err,
			"packageConfigID": fts.Integration.packageConfigID,
			"configurationID": fts.PolicyID,
		}).Warn("The integration could not be deleted from the configuration")
	}

	// clean up fields
	fts.CurrentTokenID = ""
	fts.Image = ""
	fts.Hostname = ""
}

// beforeScenario creates the state needed by a scenario
func (fts *FleetTestSuite) beforeScenario() {
	fts.Cleanup = false

	fts.Version = agentVersion

	// create policy with system monitoring enabled
	defaultPolicy, err := getAgentDefaultPolicy()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warn("The default policy could not be obtained")

		return
	}

	fts.PolicyID = defaultPolicy.Path("id").Data().(string)
}

func (fts *FleetTestSuite) contributeSteps(s *godog.ScenarioContext) {
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
	s.Step(`^certs for "([^"]*)" are installed$`, fts.installCerts)

	// endpoint steps
	s.Step(`^the "([^"]*)" integration is "([^"]*)" in the policy$`, fts.theIntegrationIsOperatedInThePolicy)
	s.Step(`^the "([^"]*)" datasource is shown in the policy as added$`, fts.thePolicyShowsTheDatasourceAdded)
	s.Step(`^the host name is shown in the Administration view in the Security App as "([^"]*)"$`, fts.theHostNameIsShownInTheAdminViewInTheSecurityApp)
	s.Step(`^the host name is not shown in the Administration view in the Security App$`, fts.theHostNameIsNotShownInTheAdminViewInTheSecurityApp)
	s.Step(`^an Endpoint is successfully deployed with a "([^"]*)" Agent using "([^"]*)" installer$`, fts.anEndpointIsSuccessfullyDeployedWithAgentAndInstalller)
	s.Step(`^the policy response will be shown in the Security App$`, fts.thePolicyResponseWillBeShownInTheSecurityApp)
	s.Step(`^the policy is updated to have "([^"]*)" in "([^"]*)" mode$`, fts.thePolicyIsUpdatedToHaveMode)
	s.Step(`^the policy will reflect the change in the Security App$`, fts.thePolicyWillReflectTheChangeInTheSecurityApp)
}

func (fts *FleetTestSuite) anStaleAgentIsDeployedToFleetWithInstaller(image, version, installerType string) error {
	agentVersionBackup := fts.Version
	defer func() { fts.Version = agentVersionBackup }()

	switch version {
	case "stale":
		version = agentStaleVersion
	case "latest":
		version = agentVersion
	default:
		version = agentStaleVersion
	}

	fts.Version = version

	// prepare installer for stale version
	if fts.Version != agentVersionBackup {
		i := GetElasticAgentInstaller(image, installerType, fts.Version, true)
		fts.Installers[fmt.Sprintf("%s-%s-%s", image, installerType, version)] = i
	}

	return fts.anAgentIsDeployedToFleetWithInstaller(image, installerType)
}

func (fts *FleetTestSuite) installCerts(targetOS string) error {
	installer := fts.getInstaller()
	if installer.InstallCertsFn == nil {
		log.WithFields(log.Fields{
			"installer":         installer,
			"version":           fts.Version,
			"agentVersion":      agentVersion,
			"agentStaleVersion": agentStaleVersion,
		}).Error("No installer found")
		return errors.New("no installer found")
	}

	return installer.InstallCertsFn()
}

func (fts *FleetTestSuite) anAgentIsUpgraded(desiredVersion string) error {
	switch desiredVersion {
	case "stale":
		desiredVersion = agentStaleVersion
	case "latest":
		desiredVersion = agentVersion
	default:
		desiredVersion = agentVersion
	}

	return fts.upgradeAgent(desiredVersion)
}

func (fts *FleetTestSuite) agentInVersion(version string) error {
	switch version {
	case "stale":
		version = agentStaleVersion
	case "latest":
		version = agentVersion
	}

	agentInVersionFn := func() error {
		agentID, err := getAgentID(fts.Hostname)
		if err != nil {
			return err
		}

		r := createDefaultHTTPRequest(fleetAgentsURL + "/" + agentID)
		body, err := curl.Get(r)
		if err != nil {
			log.WithFields(log.Fields{
				"body":  body,
				"error": err,
				"url":   r.GetURL(),
			}).Error("Could not get agent in Fleet")
			return err
		}

		jsonResponse, err := gabs.ParseJSON([]byte(body))

		retrievedVersion := jsonResponse.Path("item.local_metadata.elastic.agent.version").Data().(string)
		if isSnapshot := jsonResponse.Path("item.local_metadata.elastic.agent.snapshot").Data().(bool); isSnapshot {
			retrievedVersion += "-SNAPSHOT"
		}

		if retrievedVersion != version {
			return fmt.Errorf("version mismatch required '%s' retrieved '%s'", version, retrievedVersion)
		}

		return nil
	}

	maxTimeout := time.Duration(timeoutFactor) * time.Minute * 2
	exp := e2e.GetExponentialBackOff(maxTimeout)

	return backoff.Retry(agentInVersionFn, exp)
}

// supported installers: tar, systemd
func (fts *FleetTestSuite) anAgentIsDeployedToFleetWithInstaller(image string, installerType string) error {
	log.WithFields(log.Fields{
		"image":     image,
		"installer": installerType,
	}).Trace("Deploying an agent to Fleet with base image")

	fts.Image = image
	fts.InstallerType = installerType

	installer := fts.getInstaller()

	profile := installer.profile // name of the runtime dependencies compose file

	serviceName := ElasticAgentServiceName                                                     // name of the service
	containerName := fmt.Sprintf("%s_%s_%s_%d", profile, fts.Image+"-systemd", serviceName, 1) // name of the container

	uuid := uuid.New().String()

	// enroll the agent with a new token
	tokenJSONObject, err := createFleetToken("Test token for "+uuid, fts.PolicyID)
	if err != nil {
		return err
	}
	fts.CurrentToken = tokenJSONObject.Path("api_key").Data().(string)
	fts.CurrentTokenID = tokenJSONObject.Path("id").Data().(string)

	err = deployAgentToFleet(installer, containerName, fts.CurrentToken)
	fts.Cleanup = true
	if err != nil {
		return err
	}

	// the installation process for TAR includes the enrollment
	if installer.installerType != "tar" {
		err = installer.EnrollFn(fts.CurrentToken)
		if err != nil {
			return err
		}
	}

	// get container hostname once
	hostname, err := getContainerHostname(containerName)
	if err != nil {
		return err
	}
	fts.Hostname = hostname

	return err
}

func (fts *FleetTestSuite) getInstaller() ElasticAgentInstaller {
	return fts.Installers[fts.Image+"-"+fts.InstallerType+"-"+fts.Version]
}

func (fts *FleetTestSuite) processStateChangedOnTheHost(process string, state string) error {
	profile := FleetProfileName

	installer := fts.getInstaller()

	serviceName := installer.service // name of the service

	if state == "started" {
		return systemctlRun(profile, installer.image, serviceName, "start")
	} else if state == "restarted" {
		return systemctlRun(profile, installer.image, serviceName, "restart")
	} else if state == "uninstalled" {
		return installer.UninstallFn()
	} else if state != "stopped" {
		return godog.ErrPending
	}

	log.WithFields(log.Fields{
		"service": serviceName,
		"process": process,
	}).Trace("Stopping process on the service")

	err := systemctlRun(profile, installer.image, serviceName, "stop")
	if err != nil {
		log.WithFields(log.Fields{
			"action":  state,
			"error":   err,
			"service": serviceName,
			"process": process,
		}).Error("Could not stop process on the host")

		return err
	}

	// name of the container for the service:
	// we are using the Docker client instead of docker-compose
	// because it does not support returning the output of a
	// command: it simply returns error level
	containerName := fmt.Sprintf("%s_%s_%s_%d", profile, fts.Image+"-systemd", ElasticAgentServiceName, 1)
	return checkProcessStateOnTheHost(containerName, process, "stopped")
}

func (fts *FleetTestSuite) setup() error {
	log.Trace("Creating Fleet setup")

	err := createFleetConfiguration()
	if err != nil {
		return err
	}

	err = checkFleetConfiguration()
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) theAgentIsListedInFleetWithStatus(desiredStatus string) error {
	log.Tracef("Checking if agent is listed in Fleet as %s", desiredStatus)

	maxTimeout := time.Duration(timeoutFactor) * time.Minute * 2
	retryCount := 1

	exp := e2e.GetExponentialBackOff(maxTimeout)

	agentOnlineFn := func() error {
		agentID, err := getAgentID(fts.Hostname)
		if err != nil {
			retryCount++
			return err
		}

		if agentID == "" {
			// the agent is not listed in Fleet
			if desiredStatus == "offline" || desiredStatus == "inactive" {
				log.WithFields(log.Fields{
					"isAgentInStatus": isAgentInStatus,
					"elapsedTime":     exp.GetElapsedTime(),
					"hostname":        fts.Hostname,
					"retries":         retryCount,
					"status":          desiredStatus,
				}).Info("The Agent is not present in Fleet, as expected")
				return nil
			} else if desiredStatus == "online" {
				retryCount++
				return fmt.Errorf("The agent is not present in Fleet, but it should")
			}
		}

		isAgentInStatus, err := isAgentInStatus(agentID, desiredStatus)
		if err != nil || !isAgentInStatus {
			if err == nil {
				err = fmt.Errorf("The Agent is not in the %s status yet", desiredStatus)
			}

			log.WithFields(log.Fields{
				"agentID":         agentID,
				"isAgentInStatus": isAgentInStatus,
				"elapsedTime":     exp.GetElapsedTime(),
				"hostname":        fts.Hostname,
				"retry":           retryCount,
				"status":          desiredStatus,
			}).Warn(err.Error())

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"isAgentInStatus": isAgentInStatus,
			"elapsedTime":     exp.GetElapsedTime(),
			"hostname":        fts.Hostname,
			"retries":         retryCount,
			"status":          desiredStatus,
		}).Info("The Agent is in the desired status")
		return nil
	}

	err := backoff.Retry(agentOnlineFn, exp)
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) theFileSystemAgentFolderIsEmpty() error {
	installer := fts.getInstaller()

	profile := installer.profile // name of the runtime dependencies compose file

	// name of the container for the service:
	// we are using the Docker client instead of docker-compose
	// because it does not support returning the output of a
	// command: it simply returns error level
	containerName := fmt.Sprintf("%s_%s_%s_%d", profile, fts.Image+"-systemd", ElasticAgentServiceName, 1)

	content, err := installer.listElasticAgentWorkingDirContent(containerName)
	if err != nil {
		return err
	}

	if strings.Contains(content, "No such file or directory") {
		return nil
	}

	return fmt.Errorf("The file system directory is not empty")
}

func (fts *FleetTestSuite) theHostIsRestarted() error {
	serviceManager := services.NewServiceManager()

	installer := fts.getInstaller()

	profile := installer.profile // name of the runtime dependencies compose file
	image := installer.image     // image of the service
	service := installer.service // name of the service

	composes := []string{
		profile, // profile name
		image,   // service
	}

	err := serviceManager.RunCommand(profile, composes, []string{"restart", service}, profileEnv)
	if err != nil {
		log.WithFields(log.Fields{
			"image":   image,
			"service": service,
		}).Error("Could not restart the service")
		return err
	}

	log.WithFields(log.Fields{
		"image":   image,
		"service": service,
	}).Debug("The service has been restarted")
	return nil
}

func (fts *FleetTestSuite) systemPackageDashboardsAreListedInFleet() error {
	log.Trace("Checking system Package dashboards in Fleet")

	dataStreamsCount := 0
	maxTimeout := time.Duration(timeoutFactor) * time.Minute
	retryCount := 1

	exp := e2e.GetExponentialBackOff(maxTimeout)

	countDataStreamsFn := func() error {
		dataStreams, err := getDataStreams()
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
	return fts.unenrollHostname(false)
}

func (fts *FleetTestSuite) theAgentIsReenrolledOnTheHost() error {
	log.Trace("Re-enrolling the agent on the host with same token")

	installer := fts.getInstaller()

	err := installer.EnrollFn(fts.CurrentToken)
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

	err := fts.removeToken()
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"token":   fts.CurrentToken,
		"tokenID": fts.CurrentTokenID,
	}).Debug("Token was revoked")

	return nil
}

func (fts *FleetTestSuite) thePolicyShowsTheDatasourceAdded(packageName string) error {
	log.WithFields(log.Fields{
		"policyID": fts.PolicyID,
		"package":  packageName,
	}).Trace("Checking if the policy shows the package added")

	maxTimeout := time.Minute
	retryCount := 1

	exp := e2e.GetExponentialBackOff(maxTimeout)

	integration, err := getIntegrationFromAgentPolicy(packageName, fts.PolicyID)
	if err != nil {
		return err
	}
	fts.Integration = integration

	configurationIsPresentFn := func() error {
		defaultPolicy, err := getAgentDefaultPolicy()
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"policyID": fts.PolicyID,
				"retry":    retryCount,
			}).Warn("An error retrieving the policy happened")

			retryCount++

			return err
		}

		packagePolicies := defaultPolicy.Path("package_policies")

		for _, child := range packagePolicies.Children() {
			id := child.Data().(string)
			if id == fts.Integration.packageConfigID {
				log.WithFields(log.Fields{
					"packageConfigID": fts.Integration.packageConfigID,
					"policyID":        fts.PolicyID,
				}).Info("The integration was found in the policy")
				return nil
			}
		}

		log.WithFields(log.Fields{
			"packageConfigID": fts.Integration.packageConfigID,
			"policyID":        fts.PolicyID,
			"retry":           retryCount,
		}).Warn("The integration was not found in the policy")

		retryCount++

		return err
	}

	err = backoff.Retry(configurationIsPresentFn, exp)
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) theIntegrationIsOperatedInThePolicy(packageName string, action string) error {
	log.WithFields(log.Fields{
		"action":   action,
		"policyID": fts.PolicyID,
		"package":  packageName,
	}).Trace("Doing an operation for a package on a policy")

	if strings.ToLower(action) == actionADDED {
		name, version, err := getIntegrationLatestVersion(packageName)
		if err != nil {
			return err
		}

		integration, err := getIntegration(name, version)
		if err != nil {
			return err
		}
		fts.Integration = integration

		integrationPolicyID, err := addIntegrationToPolicy(fts.Integration, fts.PolicyID)
		if err != nil {
			return err
		}

		fts.Integration.packageConfigID = integrationPolicyID
		return nil
	} else if strings.ToLower(action) == actionREMOVED {
		integration, err := getIntegrationFromAgentPolicy(packageName, fts.PolicyID)
		if err != nil {
			return err
		}
		fts.Integration = integration

		err = deleteIntegrationFromPolicy(fts.Integration, fts.PolicyID)
		if err != nil {
			log.WithFields(log.Fields{
				"err":             err,
				"packageConfigID": fts.Integration.packageConfigID,
				"policyID":        fts.PolicyID,
			}).Error("The integration could not be deleted from the policy")
			return err
		}
		return nil
	}

	return godog.ErrPending
}

func (fts *FleetTestSuite) theHostNameIsNotShownInTheAdminViewInTheSecurityApp() error {
	log.Trace("Checking if the hostname is not shown in the Administration view in the Security App")

	maxTimeout := time.Duration(timeoutFactor) * time.Minute
	retryCount := 1

	exp := e2e.GetExponentialBackOff(maxTimeout)

	agentListedInSecurityFn := func() error {
		host, err := isAgentListedInSecurityApp(fts.Hostname)
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

		if host != nil {
			log.WithFields(log.Fields{
				"elapsedTime": exp.GetElapsedTime(),
				"host":        host,
				"hostname":    fts.Hostname,
				"retry":       retryCount,
			}).Warn("The host is still present in the Administration view in the Security App")

			retryCount++

			return fmt.Errorf("The host %s is still present in the Administration view in the Security App", fts.Hostname)
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

	maxTimeout := time.Duration(timeoutFactor) * time.Minute
	retryCount := 1

	exp := e2e.GetExponentialBackOff(maxTimeout)

	agentListedInSecurityFn := func() error {
		matches, err := isAgentListedInSecurityAppWithStatus(fts.Hostname, status)
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

func (fts *FleetTestSuite) anEndpointIsSuccessfullyDeployedWithAgentAndInstalller(image string, installer string) error {
	err := fts.anAgentIsDeployedToFleetWithInstaller(image, installer)
	if err != nil {
		return err
	}

	err = fts.theAgentIsListedInFleetWithStatus("online")
	if err != nil {
		return err
	}

	// we use integration's title
	return fts.theIntegrationIsOperatedInThePolicy(elasticEnpointIntegrationTitle, actionADDED)
}

func (fts *FleetTestSuite) thePolicyResponseWillBeShownInTheSecurityApp() error {
	agentID, err := getAgentID(fts.Hostname)
	if err != nil {
		return err
	}

	maxTimeout := time.Duration(timeoutFactor) * time.Minute
	retryCount := 1

	exp := e2e.GetExponentialBackOff(maxTimeout)

	getEventsFn := func() error {
		listed, err := isPolicyResponseListedInSecurityApp(agentID)
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

	integration, err := getIntegrationFromAgentPolicy(elasticEnpointIntegrationTitle, fts.PolicyID)
	if err != nil {
		return err
	}
	fts.Integration = integration

	integrationJSON := fts.Integration.json

	// prune fields not allowed in the API side
	prunedFields := []string{
		"created_at", "created_by", "id", "revision", "updated_at", "updated_by",
	}
	for _, f := range prunedFields {
		integrationJSON.Delete(f)
	}

	// wee only support Windows and Mac, not Linux
	integrationJSON.SetP(mode, "inputs.0.config.policy.value.windows."+name+".mode")
	integrationJSON.SetP(mode, "inputs.0.config.policy.value.mac."+name+".mode")

	response, err := updateIntegrationPackageConfig(fts.Integration.packageConfigID, integrationJSON.String())
	if err != nil {
		return err
	}

	// we use a string because we are not able to process what comes in the event, so we will do
	// an alphabetical order, as they share same layout but different millis and timezone format
	updatedAt := response.Path("item.updated_at").Data().(string)
	fts.PolicyUpdatedAt = updatedAt
	return nil
}

func (fts *FleetTestSuite) thePolicyWillReflectTheChangeInTheSecurityApp() error {
	agentID, err := getAgentID(fts.Hostname)
	if err != nil {
		return err
	}

	maxTimeout := time.Duration(timeoutFactor) * time.Minute * 2
	retryCount := 1

	exp := e2e.GetExponentialBackOff(maxTimeout)

	getEventsFn := func() error {
		err := getAgentEvents("endpoint-security", agentID, fts.Integration.packageConfigID, fts.PolicyUpdatedAt)
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

	name, version, err := getIntegrationLatestVersion(packageName)
	if err != nil {
		return err
	}

	installedIntegration, err := installIntegrationAssets(name, version)
	if err != nil {
		return err
	}
	fts.Integration = installedIntegration

	return nil
}

func (fts *FleetTestSuite) anAttemptToEnrollANewAgentFails() error {
	log.Trace("Enrolling a new agent with an revoked token")

	installer := fts.getInstaller()

	profile := installer.profile // name of the runtime dependencies compose file

	containerName := fmt.Sprintf("%s_%s_%s_%d", profile, fts.Image+"-systemd", ElasticAgentServiceName, 2) // name of the new container

	err := deployAgentToFleet(installer, containerName, fts.CurrentToken)
	// the installation process for TAR includes the enrollment
	if installer.installerType != "tar" {
		if err != nil {
			return err
		}

		err = installer.EnrollFn(fts.CurrentToken)
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

func (fts *FleetTestSuite) removeToken() error {
	revokeTokenURL := fleetEnrollmentTokenURL + "/" + fts.CurrentTokenID
	deleteReq := createDefaultHTTPRequest(revokeTokenURL)

	body, err := curl.Delete(deleteReq)
	if err != nil {
		log.WithFields(log.Fields{
			"tokenID": fts.CurrentTokenID,
			"body":    body,
			"error":   err,
			"url":     revokeTokenURL,
		}).Error("Could not delete token")
		return err
	}

	log.WithFields(log.Fields{
		"tokenID": fts.CurrentTokenID,
	}).Debug("The token was deleted")

	return nil
}

// unenrollHostname deletes the statuses for an existing agent, filtering by hostname
func (fts *FleetTestSuite) unenrollHostname(force bool) error {
	log.Tracef("Un-enrolling all agentIDs for %s", fts.Hostname)

	jsonParsed, err := getOnlineAgents(true)
	if err != nil {
		return err
	}

	hosts := jsonParsed.Path("list").Children()

	for _, host := range hosts {
		hostname := host.Path("local_metadata.host.hostname").Data().(string)
		// a hostname has an agentID by status
		if hostname == fts.Hostname {
			agentID := host.Path("id").Data().(string)
			log.WithFields(log.Fields{
				"hostname": fts.Hostname,
				"agentID":  agentID,
			}).Debug("Un-enrolling agent in Fleet")

			err := unenrollAgent(agentID, force)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (fts *FleetTestSuite) upgradeAgent(version string) error {
	agentID, err := getAgentID(fts.Hostname)
	if err != nil {
		return err
	}

	upgradeReq := curl.HTTPRequest{
		BasicAuthUser:     "elastic",
		BasicAuthPassword: "changeme",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"kbn-xsrf":     "true",
		},
		URL:     fmt.Sprintf(fleetAgentUpgradeURL, agentID),
		Payload: `{"version":"` + version + `", "force": true}`,
	}

	if content, err := curl.Post(upgradeReq); err != nil {
		return errors.Wrap(err, content)
	}

	return nil
}

// checkFleetConfiguration checks that Fleet configuration is not missing
// any requirements and is read. To achieve it, a GET request is executed
func checkFleetConfiguration() error {
	getReq := curl.HTTPRequest{
		BasicAuthUser:     "elastic",
		BasicAuthPassword: "changeme",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"kbn-xsrf":     "e2e-tests",
		},
		URL: fleetSetupURL,
	}

	log.Trace("Ensuring Fleet setup was initialised")
	responseBody, err := curl.Get(getReq)
	if err != nil {
		log.WithFields(log.Fields{
			"responseBody": responseBody,
		}).Error("Could not check Kibana setup for Fleet")
		return err
	}

	if !strings.Contains(responseBody, `"isReady":true,"missing_requirements":[]`) {
		err = fmt.Errorf("Kibana has not been initialised: %s", responseBody)
		log.Error(err.Error())
		return err
	}

	log.WithFields(log.Fields{
		"responseBody": responseBody,
	}).Info("Kibana setup initialised")

	return nil
}

// createFleetConfiguration sends a POST request to Fleet forcing the
// recreation of the configuration
func createFleetConfiguration() error {
	postReq := createDefaultHTTPRequest(fleetSetupURL)
	postReq.Payload = `{
		"forceRecreate": true
	}`

	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   fleetSetupURL,
		}).Error("Could not initialise Fleet setup")
		return err
	}

	log.WithFields(log.Fields{
		"responseBody": body,
	}).Info("Fleet setup done")

	return nil
}

// createDefaultHTTPRequest Creates a default HTTP request, including the basic auth,
// JSON content type header, and a specific header that is required by Kibana
func createDefaultHTTPRequest(url string) curl.HTTPRequest {
	return curl.HTTPRequest{
		BasicAuthUser:     "elastic",
		BasicAuthPassword: "changeme",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"kbn-xsrf":     "e2e-tests",
		},
		URL: url,
	}
}

// createFleetToken sends a POST request to Fleet creating a new token with a name
func createFleetToken(name string, policyID string) (*gabs.Container, error) {
	postReq := createDefaultHTTPRequest(fleetEnrollmentTokenURL)
	postReq.Payload = `{
		"policy_id": "` + policyID + `",
		"name": "` + name + `"
	}`

	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   fleetSetupURL,
		}).Error("Could not create Fleet token")
		return nil, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return nil, err
	}

	tokenItem := jsonParsed.Path("item")

	log.WithFields(log.Fields{
		"tokenId":  tokenItem.Path("id").Data().(string),
		"apiKeyId": tokenItem.Path("api_key_id").Data().(string),
	}).Debug("Fleet token created")

	return tokenItem, nil
}

func deployAgentToFleet(installer ElasticAgentInstaller, containerName string, token string) error {
	profile := installer.profile // name of the runtime dependencies compose file
	service := installer.service // name of the service
	serviceTag := installer.tag  // docker tag of the service

	envVarsPrefix := strings.ReplaceAll(service, "-", "_")

	// let's start with Centos 7
	profileEnv[envVarsPrefix+"Tag"] = serviceTag
	// we are setting the container name because Centos service could be reused by any other test suite
	profileEnv[envVarsPrefix+"ContainerName"] = containerName
	// define paths where the binary will be mounted
	profileEnv[envVarsPrefix+"AgentBinarySrcPath"] = installer.path
	profileEnv[envVarsPrefix+"AgentBinaryTargetPath"] = "/" + installer.name

	serviceManager := services.NewServiceManager()

	err := serviceManager.AddServicesToCompose(profile, []string{service}, profileEnv)
	if err != nil {
		log.WithFields(log.Fields{
			"service": service,
			"tag":     serviceTag,
		}).Error("Could not run the target box")
		return err
	}

	err = installer.PreInstallFn()
	if err != nil {
		return err
	}

	err = installer.InstallFn(containerName, token)
	if err != nil {
		return err
	}

	return installer.PostInstallFn()
}

// getAgentDefaultPolicy sends a GET request to Fleet for the existing default policy
func getAgentDefaultPolicy() (*gabs.Container, error) {
	r := createDefaultHTTPRequest(ingestManagerAgentPoliciesURL)
	body, err := curl.Get(r)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   ingestManagerAgentPoliciesURL,
		}).Error("Could not get Fleet's policies")
		return nil, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return nil, err
	}

	// data streams should contain array of elements
	policies := jsonParsed.Path("items")

	log.WithFields(log.Fields{
		"count": len(policies.Children()),
	}).Trace("Fleet policies retrieved")

	// TODO: perform a strong check to capture default policy
	defaultPolicy := policies.Index(0)

	return defaultPolicy, nil
}

func getAgentEvents(applicationName string, agentID string, packagePolicyID string, updatedAt string) error {
	url := fmt.Sprintf(fleetAgentEventsURL, agentID)
	getReq := createDefaultHTTPRequest(url)
	getReq.QueryString = "page=1&perPage=20"

	body, err := curl.Get(getReq)
	if err != nil {
		log.WithFields(log.Fields{
			"agentID":         agentID,
			"application":     applicationName,
			"body":            body,
			"error":           err,
			"packagePolicyID": packagePolicyID,
			"url":             url,
		}).Error("Could not get agent events from Fleet")
		return err
	}

	jsonResponse, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return err
	}

	listItems := jsonResponse.Path("list").Children()
	for _, item := range listItems {
		message := item.Path("message").Data().(string)
		// we use a string because we are not able to process what comes in the event, so we will do
		// an alphabetical order, as they share same layout but different millis and timezone format
		timestamp := item.Path("timestamp").Data().(string)

		log.WithFields(log.Fields{
			"agentID":         agentID,
			"application":     applicationName,
			"event_at":        timestamp,
			"message":         message,
			"packagePolicyID": packagePolicyID,
			"updated_at":      updatedAt,
		}).Trace("Event found")

		matches := (strings.Contains(message, applicationName) &&
			strings.Contains(message, "["+agentID+"]: State changed to") &&
			strings.Contains(message, "Protecting with policy {"+packagePolicyID+"}"))

		if matches && timestamp > updatedAt {
			log.WithFields(log.Fields{
				"application":     applicationName,
				"event_at":        timestamp,
				"packagePolicyID": packagePolicyID,
				"updated_at":      updatedAt,
				"message":         message,
			}).Info("Event after the update was found")
			return nil
		}
	}

	return fmt.Errorf("No %s events where found for the agent in the %s policy", applicationName, packagePolicyID)
}

// getAgentID sends a GET request to Fleet for a existing hostname
// This method will retrieve the only agent ID for a hostname in the online status
func getAgentID(agentHostname string) (string, error) {
	log.Tracef("Retrieving agentID for %s", agentHostname)

	jsonParsed, err := getOnlineAgents(false)
	if err != nil {
		return "", err
	}

	hosts := jsonParsed.Path("list").Children()

	for _, host := range hosts {
		hostname := host.Path("local_metadata.host.hostname").Data().(string)
		if hostname == agentHostname {
			agentID := host.Path("id").Data().(string)
			log.WithFields(log.Fields{
				"hostname": agentHostname,
				"agentID":  agentID,
			}).Debug("Agent listed in Fleet with online status")
			return agentID, nil
		}
	}

	return "", nil
}

// getDataStreams sends a GET request to Fleet for the existing data-streams
// if called prior to any Agent being deployed it should return a list of
// zero data streams as: { "data_streams": [] }. If called after the Agent
// is running, it will return a list of (currently in 7.8) 20 streams
func getDataStreams() (*gabs.Container, error) {
	r := createDefaultHTTPRequest(ingestManagerDataStreamsURL)
	body, err := curl.Get(r)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   ingestManagerDataStreamsURL,
		}).Error("Could not get Fleet's data streams for the agent")
		return nil, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return nil, err
	}

	// data streams should contain array of elements
	dataStreams := jsonParsed.Path("data_streams")

	log.WithFields(log.Fields{
		"count": len(dataStreams.Children()),
	}).Debug("Data Streams retrieved")

	return dataStreams, nil
}

// getOnlineAgents sends a GET request to Fleet for the existing online agents
// Will return the JSON object representing the response of querying Fleet's Agents
// endpoint
func getOnlineAgents(showInactive bool) (*gabs.Container, error) {
	r := createDefaultHTTPRequest(fleetAgentsURL)
	// let's not URL encode the querystring, as it seems Kibana is not handling
	// the request properly, returning an 400 Bad Request error with this message:
	// [request query.page=1&perPage=20&showInactive=true]: definition for this key is missing
	r.EncodeURL = false
	r.QueryString = fmt.Sprintf("page=1&perPage=20&showInactive=%t", showInactive)

	body, err := curl.Get(r)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   r.GetURL(),
		}).Error("Could not get Fleet's online agents")
		return nil, err
	}

	jsonResponse, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return nil, err
	}

	return jsonResponse, nil
}

// isAgentInStatus extracts the status for an agent, identified by its hostname
// It will query Fleet's agents endpoint
func isAgentInStatus(agentID string, desiredStatus string) (bool, error) {
	r := createDefaultHTTPRequest(fleetAgentsURL + "/" + agentID)
	body, err := curl.Get(r)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   r.GetURL(),
		}).Error("Could not get agent in Fleet")
		return false, err
	}

	jsonResponse, err := gabs.ParseJSON([]byte(body))

	agentStatus := jsonResponse.Path("item.status").Data().(string)

	return (strings.ToLower(agentStatus) == strings.ToLower(desiredStatus)), nil
}

func unenrollAgent(agentID string, force bool) error {
	unEnrollURL := fmt.Sprintf(fleetAgentsUnEnrollURL, agentID)
	postReq := createDefaultHTTPRequest(unEnrollURL)

	if force {
		postReq.Payload = `{
			"force": true
		}`
	}

	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"agentID": agentID,
			"body":    body,
			"error":   err,
			"url":     unEnrollURL,
		}).Error("Could unenroll agent")
		return err
	}

	log.WithFields(log.Fields{
		"agentID": agentID,
	}).Debug("Fleet agent was unenrolled")

	return nil
}
