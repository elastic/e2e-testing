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
	"github.com/cenkalti/backoff"
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

// FleetTestSuite represents the scenarios for Fleet-mode
type FleetTestSuite struct {
	AgentDownloadName string // the name for the binary
	AgentDownloadPath string // the path where the agent for the binary is installed
	EnrolledAgentID   string // will be used to store current agent
	BoxType           string // we currently support Linux
	Cleanup           bool
	ConfigID          string // will be used to manage tokens
	CurrentToken      string // current enrollment token
	CurrentTokenID    string // current enrollment tokenID
	Hostname          string // the hostname of the container
}

func (fts *FleetTestSuite) contributeSteps(s *godog.Suite) {
	s.Step(`^an agent is deployed to Fleet$`, fts.anAgentIsDeployedToFleet)
	s.Step(`^the agent is listed in Fleet as online$`, fts.theAgentIsListedInFleetAsOnline)
	s.Step(`^system package dashboards are listed in Fleet$`, fts.systemPackageDashboardsAreListedInFleet)
	s.Step(`^the agent is un-enrolled$`, fts.theAgentIsUnenrolled)
	s.Step(`^the agent is not listed as online in Fleet$`, fts.theAgentIsNotListedAsOnlineInFleet)
	s.Step(`^the agent is re-enrolled on the host$`, fts.theAgentIsReenrolledOnTheHost)
	s.Step(`^the enrollment token is revoked$`, fts.theEnrollmentTokenIsRevoked)
	s.Step(`^an attempt to enroll an agent with the old token fails$`, fts.anAttemptToEnrollANewAgentFails)
	s.Step(`^the package list API returns successfully$`, fts.thePackageListAPIReturnsSuccessfully)
	s.Step(`^the "([^"]*)" "([^"]*)" package version is installed successfully$`, fts.thePackageVersionIsInstalledSuccessfully)
	s.Step(`^the "([^"]*)" configuration shows the "([^"]*)" datasource added$`, fts.theConfigurationShowsTheDatasourceAdded)
	s.Step(`^a new configuration named "([^"]*)" is created$`, fts.aNewConfigurationNamedIsCreated)
	s.Step(`^the Agent is assigned to the configuration "([^"]*)"$`, fts.theAgentIsAssignedToTheConfiguration)
	s.Step(`^a new enrollment token is created$`, fts.aNewEnrollmentTokenIsCreated)
	s.Step(`^the "([^"]*)" package datasource is added to the "([^"]*)" configuration$`, fts.thePackageDatasourceIsAddedToTheConfiguration)
	s.Step(`^there is new data in the index from agent from "([^"]*)" stream$`, fts.thereIsNewDataInTheIndexFromAgentFromStream)
}

func (fts *FleetTestSuite) anAgentIsDeployedToFleet() error {
	log.Debug("Deploying an agent to Fleet")

	profile := "ingest-manager"                         // name of the runtime dependencies compose file
	fts.BoxType = "centos"                              // name of the service type
	serviceName := "elastic-agent"                      // name of the service
	containerName := profile + "_" + serviceName + "_1" // name of the container
	serviceTag := "7"                                   // docker tag of the service

	err := deployAgentToFleet(profile, fts.BoxType, serviceTag, containerName, fts.AgentDownloadPath, fts.AgentDownloadName)
	if err != nil {
		return err
	}
	fts.Cleanup = true

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

	err = enrollAgent(profile, fts.BoxType, serviceTag, fts.CurrentToken)
	if err != nil {
		return err
	}

	// run the agent
	err = startAgent(profile, fts.BoxType)
	if err != nil {
		return err
	}

	// get first agentID in online status, for future processing
	fts.EnrolledAgentID, err = getAgentID(true, 0)

	return err
}

// downloadAgentBinary it downloads the binary and stores the location of the downloaded file
// into the Fleet struct, to be used else where
func (fts *FleetTestSuite) downloadAgentBinary() error {
	artifact := "elastic-agent"
	version := "8.0.0-SNAPSHOT"
	os := "linux"
	arch := "x86_64"
	extension := "tar.gz"

	downloadURL, err := e2e.GetElasticArtifactURL(artifact, version, os, arch, extension)
	if err != nil {
		return err
	}

	fts.AgentDownloadName = fmt.Sprintf("%s-%s-%s-%s.%s", artifact, version, os, arch, extension)
	fts.AgentDownloadPath, err = e2e.DownloadFile(downloadURL)
	return err
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

	fts.ConfigID, err = getAgentDefaultConfig()
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) theAgentIsListedInFleetAsOnline() error {
	log.Debug("Checking agent is listed in Fleet as online")

	maxTimeout := 10 * time.Second
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

func (fts *FleetTestSuite) systemPackageDashboardsAreListedInFleet() error {
	log.Debug("Checking system Package dashboards in Fleet")

	dataStreamsCount := 0
	maxTimeout := 1 * time.Minute
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

	maxTimeout := 10 * time.Second
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

	profile := "ingest-manager"
	serviceTag := "7"

	err := enrollAgent(profile, fts.BoxType, serviceTag, fts.CurrentToken)
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

func (fts *FleetTestSuite) anAttemptToEnrollANewAgentFails() error {
	log.Debug("Enrolling a new agent with a revoked token")

	profile := "ingest-manager" // name of the runtime dependencies compose file
	serviceTag := "7"

	containerName := profile + "_" + fts.BoxType + "_2" // name of the new container

	err := deployAgentToFleet(profile, fts.BoxType, serviceTag, containerName, fts.AgentDownloadPath, fts.AgentDownloadName)
	if err != nil {
		return err
	}

	err = enrollAgent(profile, fts.BoxType, serviceTag, fts.CurrentToken)
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
		}).Error("Could delete token")
		return err
	}

	return nil
}

func (fts *FleetTestSuite) thePackageListAPIReturnsSuccessfully() error {
	return godog.ErrPending
}

func (fts *FleetTestSuite) thePackageVersionIsInstalledSuccessfully(packageName string, packageVersion string) error {
	return godog.ErrPending
}

func (fts *FleetTestSuite) theConfigurationShowsTheDatasourceAdded(configName string, packageName string) error {
	return godog.ErrPending
}

func (fts *FleetTestSuite) aNewConfigurationNamedIsCreated(configName string) error {
	return godog.ErrPending
}

func (fts *FleetTestSuite) theAgentIsAssignedToTheConfiguration(configName string) error {
	return godog.ErrPending
}

func (fts *FleetTestSuite) aNewEnrollmentTokenIsCreated() error {
	return godog.ErrPending
}

func (fts *FleetTestSuite) thePackageDatasourceIsAddedToTheConfiguration(packageName string, configName string) error {
	return godog.ErrPending
}

func (fts *FleetTestSuite) thereIsNewDataInTheIndexFromAgentFromStream(streamName string) error {
	return godog.ErrPending
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

func deployAgentToFleet(profile string, service string, serviceTag string, containerName string, agentBinaryPath string, agentBinaryName string) error {
	// let's start with Centos 7
	profileEnv[service+"Tag"] = serviceTag
	// we are setting the container name because Centos service could be reused by any other test suite
	profileEnv[service+"ContainerName"] = containerName
	// define paths where the binary will be mounted
	profileEnv[service+"AgentBinarySrcPath"] = agentBinaryPath
	profileEnv[service+"AgentBinaryTargetPath"] = "/" + agentBinaryName

	serviceManager := services.NewServiceManager()

	err := serviceManager.AddServicesToCompose(profile, []string{service}, profileEnv)
	if err != nil {
		log.WithFields(log.Fields{
			"service": service,
			"tag":     serviceTag,
		}).Error("Could not run the target box")
		return err
	}

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	version := "8.0.0-SNAPSHOT"
	os := "linux"
	arch := "x86_64"
	extension := "tar.gz"

	extractedDir := fmt.Sprintf("%s-%s-%s-%s", artifact, version, os, arch)
	tarFile := fmt.Sprintf("%s.%s", extractedDir, extension)

	cmd := []string{"tar", "xzvf", tarFile}
	err = execCommandInService(profile, service, cmd, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
			"service": service,
		}).Error("Could not extract the agent in the box")

		return err
	}

	// enable elastic-agent in PATH, because we know the location of the binary
	cmd = []string{"ln", "-s", "/" + extractedDir + "/elastic-agent", "/usr/local/bin/elastic-agent"}
	err = execCommandInService(profile, service, cmd, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
			"service": service,
		}).Error("Could not extract the agent in the box")

		return err
	}

	return nil
}

func enrollAgent(profile string, serviceName string, serviceTag string, token string) error {
	cmd := []string{"elastic-agent", "enroll", "http://kibana:5601", token, "-f", "--insecure"}
	err := execCommandInService(profile, serviceName, cmd, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
			"service": serviceName,
			"tag":     serviceTag,
			"token":   token,
		}).Error("Could not enroll the agent with the token")

		return err
	}

	return nil
}

// getAgentDefaultConfig sends a GET request to Fleet for the existing default configuration
func getAgentDefaultConfig() (string, error) {
	r := createDefaultHTTPRequest(ingestManagerAgentConfigsURL)
	body, err := curl.Get(r)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   ingestManagerAgentConfigsURL,
		}).Error("Could not get Fleet's configs")
		return "", err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return "", err
	}

	// data streams should contain array of elements
	configs := jsonParsed.Path("items")

	log.WithFields(log.Fields{
		"count": len(configs.Children()),
	}).Debug("Fleet configs retrieved")

	configID := configs.Index(0).Path("id").Data().(string)
	return configID, nil
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
		}).Error("Could not get Fleet's default enrollment token")
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
		agentStatus := agent.Path("active").Data().(bool)
		agentHostname := agent.Path("local_metadata.host.hostname").Data().(string)
		if agentHostname == hostname {
			log.WithFields(log.Fields{
				"active":   agentStatus,
				"hostname": hostname,
			}).Debug("Agent status retrieved")

			return agentStatus, nil
		}
	}

	return false, fmt.Errorf("The agent '" + hostname + "' was not found in Fleet")
}
