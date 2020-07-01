package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/cli/services"
	curl "github.com/elastic/e2e-testing/cli/shell"
	"github.com/elastic/e2e-testing/e2e"
	log "github.com/sirupsen/logrus"
)

const fleetAgentsURL = kibanaBaseURL + "/api/ingest_manager/fleet/agents?page=1&showInactive=%t"
const fleetEnrollmentTokenURL = kibanaBaseURL + "/api/ingest_manager/fleet/enrollment-api-keys"
const fleetSetupURL = kibanaBaseURL + "/api/ingest_manager/fleet/setup"
const ingestManagerDataStreamsURL = kibanaBaseURL + "/api/ingest_manager/data_streams"

// FleetTestSuite represents the scenarios for Fleet-mode
type FleetTestSuite struct {
	BoxType         string // we currently support Linux
	Cleanup         bool
	EnrollmentToken string
}

func (fts *FleetTestSuite) contributeSteps(s *godog.Suite) {
	s.Step(`^the "([^"]*)" Kibana setup has been executed$`, fts.kibanaSetupHasBeenExecuted)
	s.Step(`^an agent is deployed to Fleet$`, fts.anAgentIsDeployedToFleet)
	s.Step(`^the agent is listed in Fleet as online$`, fts.theAgentIsListedInFleetAsOnline)
	s.Step(`^system package dashboards are listed in Fleet$`, fts.systemPackageDashboardsAreListedInFleet)
	s.Step(`^there is data in the index$`, fts.thereIsDataInTheIndex)
	s.Step(`^the agent is un-enrolled$`, fts.theAgentIsUnenrolled)
	s.Step(`^the agent is not listed as online in Fleet$`, fts.theAgentIsNotListedAsOnlineInFleet)
	s.Step(`^there is no data in the index$`, fts.thereIsNoDataInTheIndex)
	s.Step(`^the agent is re-enrolled on the host$`, fts.theAgentIsReenrolledOnTheHost)
	s.Step(`^the enrollment token is revoked$`, fts.theEnrollmentTokenIsRevoked)
	s.Step(`^an attempt to enroll a new agent fails$`, fts.anAttemptToEnrollANewAgentFails)
}

func (fts *FleetTestSuite) kibanaSetupHasBeenExecuted(setup string) error {
	log.WithFields(log.Fields{
		"setup": setup,
	}).Debug("Creating Kibana setup")

	err := createFleetConfiguration()
	if err != nil {
		return err
	}

	err = checkFleetConfiguration()
	if err != nil {
		return err
	}

	token, err := getDefaultFleetEnrollmentToken()
	if err != nil {
		return err
	}

	fts.EnrollmentToken = token

	return nil
}

func (fts *FleetTestSuite) anAgentIsDeployedToFleet() error {
	log.Debug("Deploying an agent to Fleet")

	serviceManager := services.NewServiceManager()

	profile := "ingest-manager"                         // name of the runtime dependencies compose file
	fts.BoxType = "centos"                              // name of the service type
	serviceName := "elastic-agent"                      // name of the service
	containerName := profile + "_" + serviceName + "_1" // name of the container
	serviceTag := "7"                                   // docker tag of the service

	// let's start with Centos 7
	profileEnv[fts.BoxType+"Tag"] = serviceTag
	// we are setting the container name because Centos service could be reused by any other test suite
	profileEnv[fts.BoxType+"ContainerName"] = containerName

	err := serviceManager.AddServicesToCompose(profile, []string{fts.BoxType}, profileEnv)
	if err != nil {
		log.WithFields(log.Fields{
			"service": fts.BoxType,
			"tag":     serviceTag,
		}).Error("Could not run the target box")
		return err
	}

	fts.Cleanup = true

	// install the agent in the box
	cmd := []string{"curl", "-L", "-O", "https://snapshots.elastic.co/8.0.0-4c9cb790/downloads/beats/elastic-agent/elastic-agent-8.0.0-SNAPSHOT-linux-x86_64.tar.gz"}
	err = execCommandInService(profile, fts.BoxType, cmd, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
			"service": fts.BoxType,
		}).Error("Could not download agent in box")

		return err
	}

	cmd = []string{"tar", "xzvf", "elastic-agent-8.0.0-SNAPSHOT-linux-x86_64.tar.gz"}
	err = execCommandInService(profile, fts.BoxType, cmd, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
			"service": fts.BoxType,
		}).Error("Could not extract the agent in the box")

		return err
	}

	// enable elastic-agent in PATH, because we know the location of the binary
	cmd = []string{"ln", "-s", "/elastic-agent-8.0.0-SNAPSHOT-linux-x86_64/elastic-agent", "/usr/local/bin/elastic-agent"}
	err = execCommandInService(profile, fts.BoxType, cmd, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
			"service": fts.BoxType,
		}).Error("Could not extract the agent in the box")

		return err
	}

	// enroll the agent
	cmd = []string{"elastic-agent", "enroll", "http://kibana:5601", fts.EnrollmentToken, "-f"}
	err = execCommandInService(profile, fts.BoxType, cmd, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
			"service": fts.BoxType,
			"tag":     serviceTag,
		}).Error("Could not enroll the agent")

		return err
	}

	// run the agent
	cmd = []string{"elastic-agent", "run"}
	err = execCommandInService(profile, fts.BoxType, cmd, true)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
			"service": fts.BoxType,
			"tag":     serviceTag,
		}).Error("Could not run the agent")

		return err
	}

	if log.IsLevelEnabled(log.DebugLevel) {
		err = execCommandInService(profile, fts.BoxType, []string{"logs"}, false)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"service": fts.BoxType,
				"tag":     serviceTag,
			}).Error("Could not retrieve box logs")

			return err
		}
	}

	return nil
}

func (fts *FleetTestSuite) theAgentIsListedInFleetAsOnline() error {
	log.Debug("Checking agent is listed in Fleet as online")

	agentsCount, err := countAgentsByStatus(true)
	if err != nil {
		return err
	}

	if agentsCount != 1 {
		err = fmt.Errorf("There are %.0f online agents. We expected to have exactly one", agentsCount)
		log.Error(err.Error())
		return err
	}

	return nil
}

func (fts *FleetTestSuite) systemPackageDashboardsAreListedInFleet() error {
	log.Debug("Checking system Package dashboards in Fleet")

	dataStreams, err := getDataStreams()
	if err != nil {
		return err
	}

	if len(dataStreams.Children()) == 0 {
		err = fmt.Errorf("There are no datastreams. We expected to have more than one")
		log.Error(err.Error())
		return err
	}

	return nil
}

func (fts *FleetTestSuite) thereIsDataInTheIndex() error {
	log.Debug("Querying Elasticsearch index for agent data")

	return godog.ErrPending
}

func (fts *FleetTestSuite) theAgentIsUnenrolled() error {
	log.Debug("Un-enrolling agent in Fleet")

	return godog.ErrPending
}

func (fts *FleetTestSuite) theAgentIsNotListedAsOnlineInFleet() error {
	log.Debug("Checking if the agent is not listed as online in Fleet")

	return godog.ErrPending
}

func (fts *FleetTestSuite) thereIsNoDataInTheIndex() error {
	log.Debug("Querying Elasticsearch index for agent data")

	return godog.ErrPending
}

func (fts *FleetTestSuite) theAgentIsReenrolledOnTheHost() error {
	log.Debug("Re-enrolling the agent on the host")

	return godog.ErrPending
}

func (fts *FleetTestSuite) theEnrollmentTokenIsRevoked() error {
	log.Debug("Revoking enrollment token")

	return godog.ErrPending
}

func (fts *FleetTestSuite) anAttemptToEnrollANewAgentFails() error {
	log.Debug("Enrolling a new agent with an revoked token")

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

// countAgentsByStatus sends a GET request to Fleet for the existing agents
// allowing to filter by agent status: online, offline
func countAgentsByStatus(online bool) (float64, error) {
	r := createDefaultHTTPRequest(fmt.Sprintf(fleetAgentsURL, online))
	body, err := curl.Get(r)
	if err != nil {
		log.WithFields(log.Fields{
			"body":   body,
			"online": online,
			"error":  err,
			"url":    fleetAgentsURL,
		}).Error("Could not get Fleet's agents by status")
		return 0, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return 0, err
	}

	agentsCount := jsonParsed.Path("total").Data().(float64)

	log.WithFields(log.Fields{
		"count":  agentsCount,
		"online": online,
	}).Debug("Agents by status retrieved")

	return agentsCount, nil
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

func execCommandInService(profile string, serviceName string, cmds []string, detach bool) error {
	serviceManager := services.NewServiceManager()

	composes := []string{
		profile,     // profile name
		serviceName, // service
	}
	composeArgs := []string{"exec", "-T"}
	if detach {
		composeArgs = append(composeArgs, "-d")
	}
	composeArgs = append(composeArgs, serviceName)
	composeArgs = append(composeArgs, cmds...)

	err := serviceManager.RunCommand(profile, composes, composeArgs, profileEnv)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmds,
			"error":   err,
			"service": serviceName,
		}).Error("Could not execute command in container")

		return err
	}

	return nil
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

// getDefaultFleetEnrollmentToken sends a POST request to Fleet creating a new token
func getDefaultFleetEnrollmentToken() (string, error) {
	r := createDefaultHTTPRequest(fleetEnrollmentTokenURL)
	body, err := curl.Get(r)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   fleetEnrollmentTokenURL,
		}).Error("Could not get Fleet's default enrollment token")
		return "", err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return body, err
	}

	defaultTokenID := jsonParsed.Path("list").Index(0).Path("id").Data().(string)

	r = createDefaultHTTPRequest(fleetEnrollmentTokenURL + "/" + defaultTokenID)
	body, err = curl.Get(r)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   fleetEnrollmentTokenURL,
		}).Error("Could not get Fleet's default enrollment token")
		return "", err
	}

	jsonParsed, err = gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return body, err
	}

	defaultToken := jsonParsed.Path("item.api_key").Data().(string)

	log.WithFields(log.Fields{
		"token": defaultToken,
	}).Debug("Fleet default enrollment token listed")

	return defaultToken, nil
}
