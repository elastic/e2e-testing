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
	log "github.com/sirupsen/logrus"
)

const fleetAgentsURL = kibanaBaseURL + "/api/ingest_manager/fleet/agents"
const fleetAgentEventsURL = kibanaBaseURL + "/api/ingest_manager/fleet/agents/%s/events"
const fleetAgentsUnEnrollURL = kibanaBaseURL + "/api/ingest_manager/fleet/agents/%s/unenroll"
const fleetEnrollmentTokenURL = kibanaBaseURL + "/api/ingest_manager/fleet/enrollment-api-keys"
const fleetSetupURL = kibanaBaseURL + "/api/ingest_manager/fleet/setup"
const ingestManagerAgentPoliciesURL = kibanaBaseURL + "/api/ingest_manager/agent_policies"
const ingestManagerAgentPolicyURL = ingestManagerAgentPoliciesURL + "/%s"
const ingestManagerDataStreamsURL = kibanaBaseURL + "/api/ingest_manager/data_streams"

const actionADDED = "added"
const actionREMOVED = "removed"

// FleetTestSuite represents the scenarios for Fleet-mode
type FleetTestSuite struct {
	Image          string // base image used to install the agent
	Installers     map[string]ElasticAgentInstaller
	Cleanup        bool
	PolicyID       string // will be used to manage tokens
	CurrentToken   string // current enrollment token
	CurrentTokenID string // current enrollment tokenID
	Hostname       string // the hostname of the container
	// integrations
	Integration     IntegrationPackage // the installed integration
	PolicyUpdatedAt string             // the moment the policy was updated
}

func (fts *FleetTestSuite) contributeSteps(s *godog.Suite) {
	s.Step(`^a "([^"]*)" agent is deployed to Fleet$`, fts.anAgentIsDeployedToFleet)
	s.Step(`^the agent is listed in Fleet as "([^"]*)"$`, fts.theAgentIsListedInFleetWithStatus)
	s.Step(`^the host is restarted$`, fts.theHostIsRestarted)
	s.Step(`^system package dashboards are listed in Fleet$`, fts.systemPackageDashboardsAreListedInFleet)
	s.Step(`^the agent is un-enrolled$`, fts.theAgentIsUnenrolled)
	s.Step(`^the agent is re-enrolled on the host$`, fts.theAgentIsReenrolledOnTheHost)
	s.Step(`^the enrollment token is revoked$`, fts.theEnrollmentTokenIsRevoked)
	s.Step(`^an attempt to enroll a new agent fails$`, fts.anAttemptToEnrollANewAgentFails)
	s.Step(`^the "([^"]*)" process is "([^"]*)" on the host$`, fts.processStateChangedOnTheHost)

	// endpoint steps
	s.Step(`^the "([^"]*)" integration is "([^"]*)" in the "([^"]*)" policy$`, fts.theIntegrationIsOperatedInThePolicy)
	s.Step(`^the "([^"]*)" datasource is shown in the "([^"]*)" policy as added$`, fts.thePolicyShowsTheDatasourceAdded)
	s.Step(`^the host name is shown in the Administration view in the Security App as "([^"]*)"$`, fts.theHostNameIsShownInTheAdminViewInTheSecurityApp)
	s.Step(`^the host name is not shown in the Administration view in the Security App$`, fts.theHostNameIsNotShownInTheAdminViewInTheSecurityApp)
	s.Step(`^an Endpoint is successfully deployed with a "([^"]*)" Agent$`, fts.anEndpointIsSuccessfullyDeployedWithAgent)
	s.Step(`^the policy response will be shown in the Security App$`, fts.thePolicyResponseWillBeShownInTheSecurityApp)
	s.Step(`^the policy is updated to have "([^"]*)" in "([^"]*)" mode$`, fts.thePolicyIsUpdatedToHaveMode)
	s.Step(`^the policy will reflect the change in the Security App$`, fts.thePolicyWillReflectTheChangeInTheSecurityApp)
}

func (fts *FleetTestSuite) anAgentIsDeployedToFleet(image string) error {
	image = image + "-systemd" // we want to consume systemd boxes

	log.WithFields(log.Fields{
		"image": image,
	}).Trace("Deploying an agent to Fleet with base image")

	fts.Image = image

	installer := fts.Installers[fts.Image]

	profile := installer.profile // name of the runtime dependencies compose file

	serviceName := ElasticAgentServiceName                                          // name of the service
	containerName := fmt.Sprintf("%s_%s_%s_%d", profile, fts.Image, serviceName, 1) // name of the container

	err := deployAgentToFleet(installer, containerName)
	fts.Cleanup = true
	if err != nil {
		return err
	}

	// get container hostname once
	hostname, err := getContainerHostname(containerName)
	if err != nil {
		return err
	}
	fts.Hostname = hostname

	// enroll the agent with a new token
	tokenJSONObject, err := createFleetToken("Test token for "+hostname, fts.PolicyID)
	if err != nil {
		return err
	}
	fts.CurrentToken = tokenJSONObject.Path("api_key").Data().(string)
	fts.CurrentTokenID = tokenJSONObject.Path("id").Data().(string)

	err = enrollAgent(installer, fts.CurrentToken)
	if err != nil {
		return err
	}

	err = systemctlRun(profile, image, image, "start")
	if err != nil {
		return err
	}

	return err
}

func (fts *FleetTestSuite) processStateChangedOnTheHost(process string, state string) error {
	profile := IngestManagerProfileName
	image := fts.Image

	installer := fts.Installers[fts.Image]

	serviceName := installer.service // name of the service

	if state == "started" {
		return systemctlRun(profile, image, serviceName, "start")
	} else if state != "stopped" {
		return godog.ErrPending
	}

	log.WithFields(log.Fields{
		"service": serviceName,
		"process": process,
	}).Trace("Stopping process on the service")

	err := systemctlRun(profile, image, serviceName, "stop")
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
	containerName := fmt.Sprintf("%s_%s_%s_%d", profile, fts.Image, ElasticAgentServiceName, 1)
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

	defaultPolicy, err := getAgentDefaultPolicy()
	if err != nil {
		return err
	}
	fts.PolicyID = defaultPolicy.Path("id").Data().(string)

	return nil
}

func (fts *FleetTestSuite) theAgentIsListedInFleetWithStatus(desiredStatus string) error {
	log.Tracef("Checking if agent is listed in Fleet as %s", desiredStatus)

	maxTimeout := 2 * time.Minute
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
			if desiredStatus == "inactive" {
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

func (fts *FleetTestSuite) theHostIsRestarted() error {
	serviceManager := services.NewServiceManager()

	installer := fts.Installers[fts.Image]

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
	maxTimeout := 2 * time.Minute
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

	installer := fts.Installers[fts.Image]

	err := enrollAgent(installer, fts.CurrentToken)
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

func (fts *FleetTestSuite) thePolicyShowsTheDatasourceAdded(packageName string, policyName string) error {
	log.WithFields(log.Fields{
		"policy":  policyName,
		"package": packageName,
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

func (fts *FleetTestSuite) theIntegrationIsOperatedInThePolicy(packageName string, action string, policyName string) error {
	log.WithFields(log.Fields{
		"action":  action,
		"policy":  policyName,
		"package": packageName,
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

	maxTimeout := 2 * time.Minute
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

	maxTimeout := 2 * time.Minute
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

func (fts *FleetTestSuite) anEndpointIsSuccessfullyDeployedWithAgent(image string) error {
	err := fts.anAgentIsDeployedToFleet(image)
	if err != nil {
		return err
	}

	err = fts.theAgentIsListedInFleetWithStatus("online")
	if err != nil {
		return err
	}

	// we use integration's title
	return fts.theIntegrationIsOperatedInThePolicy("Elastic Endpoint", actionADDED, "default")
}

func (fts *FleetTestSuite) thePolicyResponseWillBeShownInTheSecurityApp() error {
	agentID, err := getAgentID(fts.Hostname)
	if err != nil {
		return err
	}

	maxTimeout := 2 * time.Minute
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

	integration, err := getIntegrationFromAgentPolicy("Elastic Endpoint", fts.PolicyID)
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
	// an alphabetical order, as they share same layour but different millis and timezone format
	updatedAt := response.Path("item.updated_at").Data().(string)
	fts.PolicyUpdatedAt = updatedAt
	return nil
}

func (fts *FleetTestSuite) thePolicyWillReflectTheChangeInTheSecurityApp() error {
	agentID, err := getAgentID(fts.Hostname)
	if err != nil {
		return err
	}

	maxTimeout := 2 * time.Minute
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

	installer := fts.Installers[fts.Image]

	profile := installer.profile // name of the runtime dependencies compose file
	service := installer.service // name of the service

	containerName := fmt.Sprintf("%s_%s_%s_%d", profile, fts.Image, service, 2) // name of the new container

	err := deployAgentToFleet(installer, containerName)
	if err != nil {
		return err
	}

	err = enrollAgent(installer, fts.CurrentToken)
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

func deployAgentToFleet(installer ElasticAgentInstaller, containerName string) error {
	profile := installer.profile // name of the runtime dependencies compose file
	image := installer.image     // image of the service
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

	cmd := installer.InstallCmds
	err = execCommandInService(profile, image, service, cmd, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
			"image":   image,
			"service": service,
		}).Error("Could not install the agent in the box")

		return err
	}

	return installer.PostInstallFn()
}

func enrollAgent(installer ElasticAgentInstaller, token string) error {
	profile := installer.profile // name of the runtime dependencies compose file
	image := installer.image     // image of the service
	service := installer.service // name of the service
	serviceTag := installer.tag  // tag of the service

	cmd := []string{installer.processName, "enroll", "http://kibana:5601", token, "-f", "--insecure"}
	err := execCommandInService(profile, image, service, cmd, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
			"image":   image,
			"service": service,
			"tag":     serviceTag,
			"token":   token,
		}).Error("Could not enroll the agent with the token")

		return err
	}

	return nil
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
