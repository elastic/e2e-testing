package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/cucumber/godog"
	"github.com/cucumber/messages-go/v10"
	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/cli/services"
	curl "github.com/elastic/e2e-testing/cli/shell"
	"github.com/elastic/e2e-testing/e2e"
	log "github.com/sirupsen/logrus"
)

// stackVersion is the version of the stack to use
// It can be overriden by OP_STACK_VERSION env var
var stackVersion = "7.7.0"

func init() {
	config.Init()

	stackVersion = e2e.GetEnv("OP_STACK_VERSION", stackVersion)
}

func IngestManagerFeatureContext(s *godog.Suite) {
	imts := IngestManagerTestSuite{}
	serviceManager := services.NewServiceManager()

	s.Step(`^the "([^"]*)" Kibana setup has been executed$`, imts.kibanaSetupHasBeenExecuted)
	s.Step(`^an agent is deployed to Fleet$`, imts.anAgentIsDeployedToFleet)
	s.Step(`^the agent is listed in Fleet as online$`, imts.theAgentIsListedInFleetAsOnline)
	s.Step(`^system package dashboards are listed in Fleet$`, imts.systemPackageDashboardsAreListedInFleet)
	s.Step(`^there is data in the index$`, imts.thereIsDataInTheIndex)
	s.Step(`^the "([^"]*)" process is "([^"]*)" on the host$`, imts.processStateOnTheHost)
	s.Step(`^the agent is un-enrolled$`, imts.theAgentIsUnenrolled)
	s.Step(`^the agent is not listed as online in Fleet$`, imts.theAgentIsNotListedAsOnlineInFleet)
	s.Step(`^there is no data in the index$`, imts.thereIsNoDataInTheIndex)
	s.Step(`^an agent is enrolled$`, imts.anAgentIsEnrolled)
	s.Step(`^the agent is re-enrolled on the host$`, imts.theAgentIsReenrolledOnTheHost)
	s.Step(`^the enrollment token is revoked$`, imts.theEnrollmentTokenIsRevoked)
	s.Step(`^an attempt to enroll a new agent fails$`, imts.anAttemptToEnrollANewAgentFails)

	s.BeforeSuite(func() {
		log.Debug("Installing ingest-manager runtime dependencies")

		workDir, _ := os.Getwd()
		env := map[string]string{
			"stackVersion":     stackVersion,
			"kibanaConfigPath": path.Join(workDir, "configurations", "kibana.config.yml"),
		}

		profile := "ingest-manager"
		err := serviceManager.RunCompose(true, []string{profile}, env)
		if err != nil {
			log.WithFields(log.Fields{
				"profile": profile,
			}).Error("Could not run the runtime dependencies for the profile.")
		}

		minutesToBeHealthy := 3 * time.Minute
		healthy, err := e2e.WaitForElasticsearch(minutesToBeHealthy)
		if !healthy {
			log.WithFields(log.Fields{
				"error":   err,
				"minutes": minutesToBeHealthy,
			}).Error("The Elasticsearch cluster could not get the healthy status")
		}

		healthyKibana, err := e2e.WaitForKibana(minutesToBeHealthy)
		if !healthyKibana {
			log.WithFields(log.Fields{
				"error":   err,
				"minutes": minutesToBeHealthy,
			}).Error("The Kibana instance could not get the healthy status")
		}
	})
	s.BeforeScenario(func(*messages.Pickle) {
		log.Debug("Before Ingest Manager scenario")

		imts.CleanupAgent = false
	})
	s.AfterSuite(func() {
		log.Debug("Destroying ingest-manager runtime dependencies")
		profile := "ingest-manager"

		err := serviceManager.StopCompose(true, []string{profile})
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"profile": profile,
			}).Warn("Could not destroy the runtime dependencies for the profile.")
		}
	})
	s.AfterScenario(func(*messages.Pickle, error) {
		log.Debug("After Ingest Manager scenario")

		if imts.CleanupAgent {
			serviceName := "elastic-agent"
			env := map[string]string{}

			services := []string{serviceName}

			err := serviceManager.RemoveServicesFromCompose("ingest-manager", services, env)
			if err != nil {
				log.WithFields(log.Fields{
					"service": serviceName,
				}).Error("Could not stop the service.")
			}

			log.WithFields(log.Fields{
				"service": serviceName,
			}).Debug("Service removed from compose.")
		}
	})
}

// IngestManagerTestSuite represents a test suite, holding references to the pieces needed to run the tests
type IngestManagerTestSuite struct {
	CleanupAgent    bool
	EnrollmentToken string
}

func (imts *IngestManagerTestSuite) kibanaSetupHasBeenExecuted(setup string) error {
	log.WithFields(log.Fields{
		"setup": setup,
	}).Debug("Creating Kibana setup")

	// running on localhost as Kibana is expected to be exposed there
	fleetSetupURL := "http://localhost:5601/api/ingest_manager/fleet/setup"

	err := createFleetConfiguration(fleetSetupURL)
	if err != nil {
		return err
	}

	err = checkFleetConfiguration(fleetSetupURL)
	if err != nil {
		return err
	}

	fleetEnrollmentTokenURL := "http://localhost:5601/api/ingest_manager/fleet/enrollment-api-keys"

	token, err := getDefaultFleetEnrollmentToken(fleetEnrollmentTokenURL)
	if err != nil {
		return err
	}

	imts.EnrollmentToken = token

	return nil
}

func (imts *IngestManagerTestSuite) anAgentIsDeployedToFleet() error {
	log.Debug("Deploying an agent to Fleet")

	serviceManager := services.NewServiceManager()

	profile := "ingest-manager"
	serviceName := "elastic-agent"

	env := map[string]string{}

	err := serviceManager.AddServicesToCompose(profile, []string{serviceName}, env)
	if err != nil {
		log.Error("Could not deploy the elastic-agent")
		return err
	}

	imts.CleanupAgent = true

	// enroll an agent
	composes := []string{
		profile,     // profile name
		serviceName, // agent service
	}
	composeArgs := []string{"exec", "-d", "-T", serviceName}
	cmd := []string{"elastic-agent", "enroll", "http://localhost:5601", imts.EnrollmentToken, "-f"}
	composeArgs = append(composeArgs, cmd...)

	err = serviceManager.RunCommand(profile, composes, composeArgs, env)
	if err != nil {
		log.WithFields(log.Fields{
			"command":     cmd,
			"error":       err,
			"serviceName": serviceName,
		}).Error("Could not execute command in agent")

		return err
	}

	return nil
}

func (imts *IngestManagerTestSuite) theAgentIsListedInFleetAsOnline() error {
	log.Debug("Checking agent is listed in Fleet as online")

	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) systemPackageDashboardsAreListedInFleet() error {
	log.Debug("Checking system Package dashboards in Fleet")

	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) thereIsDataInTheIndex() error {
	log.Debug("Querying Elasticsearch index for agent data")

	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) processStateOnTheHost(process string, state string) error {
	log.WithFields(log.Fields{
		"process": process,
		"state":   state,
	}).Debug("Checking process state on the host")

	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) theAgentIsUnenrolled() error {
	log.Debug("Un-enrolling agent in Fleet")

	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) theAgentIsNotListedAsOnlineInFleet() error {
	log.Debug("Checking if the agent is not listed as online in Fleet")

	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) thereIsNoDataInTheIndex() error {
	log.Debug("Querying Elasticsearch index for agent data")

	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) anAgentIsEnrolled() error {
	log.Debug("Enrolling an agent")

	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) theAgentIsReenrolledOnTheHost() error {
	log.Debug("Re-enrolling the agent on the host")

	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) theEnrollmentTokenIsRevoked() error {
	log.Debug("Revoking enrollment token")

	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) anAttemptToEnrollANewAgentFails() error {
	log.Debug("Enrolling a new agent with an revoked token")

	return godog.ErrPending
}

// checkFleetConfiguration checks that Fleet configuration is not missing
// any requirements and is read. To achieve it, a GET request is executed
func checkFleetConfiguration(fleetSetupURL string) error {
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

// createFleetConfiguration sends a POST request to Fleet forcing the
// recreation of the configuration
func createFleetConfiguration(fleetSetupURL string) error {
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

// getDefaultFleetEnrollmentToken sends a POST request to Fleet creating a new token
func getDefaultFleetEnrollmentToken(fleetEnrollmentTokenURL string) (string, error) {
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
