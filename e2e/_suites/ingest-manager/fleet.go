// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"encoding/json"
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
const fleetAgentsUnEnrollURL = kibanaBaseURL + "/api/ingest_manager/fleet/agents/%s/unenroll"
const fleetEnrollmentTokenURL = kibanaBaseURL + "/api/ingest_manager/fleet/enrollment-api-keys"
const fleetSetupURL = kibanaBaseURL + "/api/ingest_manager/fleet/setup"
const ingestManagerAgentConfigsURL = kibanaBaseURL + "/api/ingest_manager/agent_configs"
const ingestManagerDataStreamsURL = kibanaBaseURL + "/api/ingest_manager/data_streams"

const actionADDED = "added"
const actionREMOVED = "removed"

// FleetTestSuite represents the scenarios for Fleet-mode
type FleetTestSuite struct {
	EnrolledAgentID string // will be used to store current agent
	Image           string // base image used to install the agent
	Installers      map[string]ElasticAgentInstaller
	Cleanup         bool
	ConfigID        string // will be used to manage tokens
	CurrentToken    string // current enrollment token
	CurrentTokenID  string // current enrollment tokenID
	Hostname        string // the hostname of the container
	// integrations
	Integration IntegrationPackage // the installed integration
}

func (fts *FleetTestSuite) contributeSteps(s *godog.Suite) {
	s.Step(`^a "([^"]*)" agent is deployed to Fleet$`, fts.anAgentIsDeployedToFleet)
	s.Step(`^the agent is listed in Fleet as online$`, fts.theAgentIsListedInFleetAsOnline)
	s.Step(`^the host is restarted$`, fts.theHostIsRestarted)
	s.Step(`^system package dashboards are listed in Fleet$`, fts.systemPackageDashboardsAreListedInFleet)
	s.Step(`^the agent is un-enrolled$`, fts.theAgentIsUnenrolled)
	s.Step(`^the agent is not listed as online in Fleet$`, fts.theAgentIsNotListedAsOnlineInFleet)
	s.Step(`^the agent is re-enrolled on the host$`, fts.theAgentIsReenrolledOnTheHost)
	s.Step(`^the enrollment token is revoked$`, fts.theEnrollmentTokenIsRevoked)
	s.Step(`^an attempt to enroll a new agent fails$`, fts.anAttemptToEnrollANewAgentFails)
	s.Step(`^the "([^"]*)" process is "([^"]*)" on the host$`, fts.processStateChangedOnTheHost)

	// endpoint steps
	s.Step(`^the "([^"]*)" version of the "([^"]*)" package is installed$`, fts.theVersionOfThePackageIsInstalled)
	s.Step(`^the "([^"]*)" integration is "([^"]*)" in the "([^"]*)" configuration$`, fts.theIntegrationIsOperatedInTheConfiguration)
	s.Step(`^the "([^"]*)" datasource is shown in the "([^"]*)" configuration as added$`, fts.theConfigurationShowsTheDatasourceAdded)
	s.Step(`^the host name is shown in the Security App$`, fts.theHostNameIsShownInTheSecurityApp)
	s.Step(`^an Endpoint is successfully deployed with Agent$`, fts.anEndpointIsSuccessfullyDeployedWithAgent)
	s.Step(`^the policy response will be shown in the Security App$`, fts.thePolicyResponseWillBeShownInTheSecurityApp)
	s.Step(`^the policy is updated to have malware in detect mode$`, fts.thePolicyIsUpdatedToHaveMalwareInDetectMode)
	s.Step(`^the policy will reflect the change in the Security App$`, fts.thePolicyWillReflectTheChangeInTheSecurityApp)
	s.Step(`^the endpoint is not listed in Security App as online$`, fts.theEndpointIsNotListedInSecurityAppAsOnline)
}

func (fts *FleetTestSuite) anAgentIsDeployedToFleet(image string) error {
	image = image + "-systemd" // we want to consume systemd boxes

	log.WithFields(log.Fields{
		"image": image,
	}).Debug("Deploying an agent to Fleet with base image")

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
	tokenJSONObject, err := createFleetToken("Test token for "+hostname, fts.ConfigID)
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

	// get first agentID in online status, for future processing
	fts.EnrolledAgentID, err = getAgentID(true, 0)

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
	}).Debug("Stopping process on the service")

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
	log.Debug("Creating Fleet setup")

	err := createFleetConfiguration()
	if err != nil {
		return err
	}

	err = checkFleetConfiguration()
	if err != nil {
		return err
	}

	defaultConfig, err := getAgentDefaultConfig()
	if err != nil {
		return err
	}
	fts.ConfigID = defaultConfig.Path("id").Data().(string)

	return nil
}

func (fts *FleetTestSuite) theAgentIsListedInFleetAsOnline() error {
	log.Debug("Checking agent is listed in Fleet as online")

	maxTimeout := 2 * time.Minute
	retryCount := 1

	exp := e2e.GetExponentialBackOff(maxTimeout)

	agentOnlineFn := func() error {
		status, err := isAgentOnline(fts.Hostname)
		if err != nil || !status {
			if err == nil {
				err = fmt.Errorf("The Agent is not online yet")
			}

			log.WithFields(log.Fields{
				"active":      status,
				"elapsedTime": exp.GetElapsedTime(),
				"hostname":    fts.Hostname,
				"retry":       retryCount,
			}).Warn(err.Error())

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"active":      status,
			"elapsedTime": exp.GetElapsedTime(),
			"hostname":    fts.Hostname,
			"retries":     retryCount,
		}).Info("The Agent is online")
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
	log.Debug("Checking system Package dashboards in Fleet")

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
	log.WithFields(log.Fields{
		"agentID": fts.EnrolledAgentID,
	}).Debug("Un-enrolling agent in Fleet")

	unEnrollURL := fmt.Sprintf(fleetAgentsUnEnrollURL, fts.EnrolledAgentID)
	postReq := createDefaultHTTPRequest(unEnrollURL)

	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"agentID": fts.EnrolledAgentID,
			"body":    body,
			"error":   err,
			"url":     unEnrollURL,
		}).Error("Could unenroll agent")
		return err
	}

	log.WithFields(log.Fields{
		"agentID": fts.EnrolledAgentID,
	}).Debug("Fleet agent was unenrolled")

	return nil
}

func (fts *FleetTestSuite) theAgentIsNotListedAsOnlineInFleet() error {
	log.Debug("Checking if the agent is not listed as online in Fleet")

	maxTimeout := 2 * time.Minute
	retryCount := 1

	exp := e2e.GetExponentialBackOff(maxTimeout)

	agentOnlineFn := func() error {
		status, err := isAgentOnline(fts.Hostname)
		if err != nil || status {
			if err == nil {
				err = fmt.Errorf("The Agent is still online")
			}

			log.WithFields(log.Fields{
				"active":      status,
				"elapsedTime": exp.GetElapsedTime(),
				"hostname":    fts.Hostname,
				"retry":       retryCount,
			}).Warn(err.Error())

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"active":      status,
			"elapsedTime": exp.GetElapsedTime(),
			"hostname":    fts.Hostname,
			"retries":     retryCount,
		}).Info("The Agent is offline")
		return nil
	}

	err := backoff.Retry(agentOnlineFn, exp)
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) theAgentIsReenrolledOnTheHost() error {
	log.Debug("Re-enrolling the agent on the host with same token")

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
	}).Debug("Revoking enrollment token")

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

func (fts *FleetTestSuite) theConfigurationShowsTheDatasourceAdded(configurationName string, packageName string) error {
	log.WithFields(log.Fields{
		"configuration": configurationName,
		"package":       packageName,
	}).Debug("Checking if the configuration shows the package added")

	maxTimeout := time.Minute
	retryCount := 1

	exp := e2e.GetExponentialBackOff(maxTimeout)

	configurationIsPresentFn := func() error {
		defaultConfig, err := getAgentDefaultConfig()
		if err != nil {
			log.WithFields(log.Fields{
				"error":           err,
				"packageConfigID": fts.Integration.packageConfigID,
				"configurationID": fts.ConfigID,
				"retry":           retryCount,
			}).Warn("An error retrieving the configuration happened")

			retryCount++

			return err
		}

		packageConfigs := defaultConfig.Path("package_configs")

		for _, child := range packageConfigs.Children() {
			id := child.Data().(string)
			if id == fts.Integration.packageConfigID {
				log.WithFields(log.Fields{
					"packageConfigID": fts.Integration.packageConfigID,
					"configurationID": fts.ConfigID,
				}).Info("The integration was found in the configuration")
				return nil
			}
		}

		log.WithFields(log.Fields{
			"packageConfigID": fts.Integration.packageConfigID,
			"configurationID": fts.ConfigID,
			"retry":           retryCount,
		}).Warn("The integration was not found in the configuration")

		retryCount++

		return err
	}

	err := backoff.Retry(configurationIsPresentFn, exp)
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) theIntegrationIsOperatedInTheConfiguration(packageName string, action string, configurationName string) error {
	log.WithFields(log.Fields{
		"action":        action,
		"configuration": configurationName,
		"package":       packageName,
	}).Debug("Doing an operation for a package on a configuration")

	if strings.ToLower(action) == actionADDED {
		integrationConfigurationID, err := addIntegrationToConfiguration(fts.Integration, fts.ConfigID)
		if err != nil {
			return err
		}

		fts.Integration.packageConfigID = integrationConfigurationID
		return nil
	} else if strings.ToLower(action) == actionREMOVED {
		err := deleteIntegrationFromConfiguration(fts.Integration, fts.ConfigID)
		if err != nil {
			log.WithFields(log.Fields{
				"err":             err,
				"packageConfigID": fts.Integration.packageConfigID,
				"configurationID": fts.ConfigID,
			}).Error("The integration could not be deleted from the configuration")
			return err
		}
		return nil
	}

	return godog.ErrPending
}

func (fts *FleetTestSuite) theHostNameIsShownInTheSecurityApp() error {
	log.Debug("Checking if the hostname is shown in the Security App")

	return godog.ErrPending
}

func (fts *FleetTestSuite) anEndpointIsSuccessfullyDeployedWithAgent() error {
	err := fts.anAgentIsDeployedToFleet("centos")
	if err != nil {
		return err
	}

	err = fts.theVersionOfThePackageIsInstalled("latest", "endpoint")
	if err != nil {
		return err
	}

	err = fts.theIntegrationIsOperatedInTheConfiguration("enpdoint", actionADDED, "default")
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) thePolicyResponseWillBeShownInTheSecurityApp() error {
	return godog.ErrPending
}

func (fts *FleetTestSuite) thePolicyIsUpdatedToHaveMalwareInDetectMode() error {
	return godog.ErrPending
}

func (fts *FleetTestSuite) thePolicyWillReflectTheChangeInTheSecurityApp() error {
	return godog.ErrPending
}

func (fts *FleetTestSuite) theEndpointIsNotListedInSecurityAppAsOnline() error {
	return godog.ErrPending
}

func (fts *FleetTestSuite) theVersionOfThePackageIsInstalled(version string, packageName string) error {
	log.WithFields(log.Fields{
		"package": packageName,
		"version": version,
	}).Debug("Checking if package version is installed")

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
	log.Debug("Enrolling a new agent with an revoked token")

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

	log.Debug("Ensuring Fleet setup was initialised")
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
	type payload struct {
		ForceRecreate bool `json:"forceRecreate"`
	}

	data := payload{
		ForceRecreate: true,
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		log.Error("Could not serialise payload")
		return err
	}

	postReq := createDefaultHTTPRequest(fleetSetupURL)

	postReq.Payload = payloadBytes

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
	}).Debug("Fleet setup done")

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
func createFleetToken(name string, configID string) (*gabs.Container, error) {
	type payload struct {
		ConfigID string `json:"config_id"`
		Name     string `json:"name"`
	}

	data := payload{
		ConfigID: configID,
		Name:     name,
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		log.Error("Could not serialise payload")
		return nil, err
	}

	postReq := createDefaultHTTPRequest(fleetEnrollmentTokenURL)

	postReq.Payload = payloadBytes

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

// getAgentDefaultConfig sends a GET request to Fleet for the existing default configuration
func getAgentDefaultConfig() (*gabs.Container, error) {
	r := createDefaultHTTPRequest(ingestManagerAgentConfigsURL)
	body, err := curl.Get(r)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   ingestManagerAgentConfigsURL,
		}).Error("Could not get Fleet's configs")
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
	configs := jsonParsed.Path("items")

	log.WithFields(log.Fields{
		"count": len(configs.Children()),
	}).Debug("Fleet configs retrieved")

	// TODO: perform a strong check to capture default config
	defaultConfig := configs.Index(0)

	return defaultConfig, nil
}

// getAgentID sends a GET request to Fleet for the existing agents
// allowing to filter by agent status: online, offline. This method will
// retrieve the agent ID
func getAgentID(online bool, index int) (string, error) {
	jsonParsed, err := getOnlineAgents()
	if err != nil {
		return "", err
	}

	agentID := jsonParsed.Path("list").Index(index).Path("id").Data().(string)

	log.WithFields(log.Fields{
		"index":   index,
		"agentID": agentID,
	}).Debug("Agent ID retrieved")

	return agentID, nil
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

// getAgentsByStatus sends a GET request to Fleet for the existing online agents
// Will return the JSON object representing the response of querying Fleet's Agents
// endpoint
func getOnlineAgents() (*gabs.Container, error) {
	r := createDefaultHTTPRequest(fleetAgentsURL)
	// let's not URL encode the querystring, as it seems Kibana is not handling
	// the request properly, returning an 400 Bad Request error with this message:
	// [request query.page=1&perPage=20&showInactive=true]: definition for this key is missing
	r.EncodeURL = false
	r.QueryString = fmt.Sprintf("page=1&perPage=20&showInactive=%t", true)

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

// isAgentOnline extracts the status for an agent, identified by its hotname
// It will wuery Fleet's agents endpoint
func isAgentOnline(hostname string) (bool, error) {
	jsonResponse, err := getOnlineAgents()
	if err != nil {
		return false, err
	}

	agents := jsonResponse.Path("list")

	for _, agent := range agents.Children() {
		agentStatus := agent.Path("status").Data().(string)
		agentHostname := agent.Path("local_metadata.host.hostname").Data().(string)

		log.WithFields(log.Fields{
			"status":   agentStatus,
			"hostname": agentHostname,
		}).Debug("Agent status retrieved")

		if agentHostname == hostname {
			isOnline := (strings.ToLower(agentStatus) == "online")
			return isOnline, nil
		}
	}

	return false, fmt.Errorf("The agent '" + hostname + "' was not found in Fleet")
}
