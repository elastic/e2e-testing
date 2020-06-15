package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/cli/services"
	curl "github.com/elastic/e2e-testing/cli/shell"
	log "github.com/sirupsen/logrus"
)

const fleetAgentsURL = kibanaBaseURL + "/api/ingest_manager/fleet/agents?page=1&showInactive=%t"
const fleetEnrollmentTokenURL = kibanaBaseURL + "/api/ingest_manager/fleet/enrollment-api-keys"
const fleetSetupURL = kibanaBaseURL + "/api/ingest_manager/fleet/setup"
const ingestManagerDataStreamsURL = kibanaBaseURL + "/api/ingest_manager/data_streams"

// FleetTestSuite represents the scenarios for Fleet-mode
type FleetTestSuite struct {
	CleanupAgent    bool
	EnrollmentToken string
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

	profile := "ingest-manager"
	serviceName := "elastic-agent"

	err := serviceManager.AddServicesToCompose(profile, []string{serviceName}, profileEnv)
	if err != nil {
		log.Error("Could not deploy the elastic-agent")
		return err
	}

	fts.CleanupAgent = true

	// enroll an agent
	composes := []string{
		profile,     // profile name
		serviceName, // agent service
	}
	composeArgs := []string{"exec", "-d", "-T", serviceName}
	cmd := []string{"elastic-agent", "enroll", "http://localhost:5601", fts.EnrollmentToken, "-f"}
	composeArgs = append(composeArgs, cmd...)

	err = serviceManager.RunCommand(profile, composes, composeArgs, profileEnv)
	if err != nil {
		log.WithFields(log.Fields{
			"command":     cmd,
			"error":       err,
			"serviceName": serviceName,
		}).Error("Could not execute command in agent")

		return err
	}

	if log.IsLevelEnabled(log.DebugLevel) {
		err = serviceManager.RunCommand(profile, composes, []string{"logs", serviceName}, profileEnv)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"service": serviceName,
			}).Error("Could not retrieve Elastic Agent logs")

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

func (fts *FleetTestSuite) processStateOnTheHost(process string, state string) error {
	log.WithFields(log.Fields{
		"process": process,
		"state":   state,
	}).Debug("Checking process state on the host")

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
		BasicAuthPassword: "p4ssw0rd",
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
		BasicAuthPassword: "p4ssw0rd",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"kbn-xsrf":     "e2e-tests",
		},
		URL: url,
	}
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
