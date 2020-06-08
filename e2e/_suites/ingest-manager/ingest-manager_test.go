package main

import (
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/messages-go/v10"
	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/cli/services"
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

	s.Step(`^the "([^"]*)" Kibana setup has been executed$`, imts.kibanaSetupHasBeenExecuted)
	s.Step(`^an agent is deployed to Fleet$`, imts.anAgentIsDeployedToFleet)
	s.Step(`^the agent is listed in Fleet as online$`, imts.theAgentIsListedInFleetAsOnline)
	s.Step(`^system package dashboards are listed in Fleet$`, imts.systemPackageDashboardsAreListedInFleet)
	s.Step(`^there is data in the index$`, imts.thereIsDataInTheIndex)
	s.Step(`^"([^"]*)" is "([^"]*)" on the host$`, imts.processStateOnTheHost)
	s.Step(`^the agent is "([^"]*)" on the host$`, imts.agentStateOnTheHost)
	s.Step(`^the agent is un-enrolled$`, imts.theAgentIsUnenrolled)
	s.Step(`^the agent is not listed as online in Fleet$`, imts.theAgentIsNotListedAsOnlineInFleet)
	s.Step(`^there is no data in the index$`, imts.thereIsNoDataInTheIndex)
	s.Step(`^an agent is enrolled$`, imts.anAgentIsEnrolled)
	s.Step(`^the agent is re-enrolled on the host$`, imts.theAgentIsReenrolledOnTheHost)
	s.Step(`^"([^"]*)" is listed in Fleet as online$`, imts.isListedInFleetAsOnline)
	s.Step(`^the enrollment token is revoked$`, imts.theEnrollmentTokenIsRevoked)
	s.Step(`^an attempt to enroll a new agent fails$`, imts.anAttemptToEnrollANewAgentFails)

	s.BeforeSuite(func() {
		log.Debug("Installing ingest-manager runtime dependencies")
		serviceManager := services.NewServiceManager()

		env := map[string]string{
			"stackVersion": stackVersion,
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
	})
	s.BeforeScenario(func(*messages.Pickle) {
		log.Debug("Before Ingest Manager scenario")
	})
	s.AfterSuite(func() {
		log.Debug("Destroying ingest-manager runtime dependencies")
		serviceManager := services.NewServiceManager()
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
	})
}

// IngestManagerTestSuite represents a test suite, holding references to the pieces needed to run the tests
type IngestManagerTestSuite struct {
}

func (imts *IngestManagerTestSuite) kibanaSetupHasBeenExecuted(setup string) error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) anAgentIsDeployedToFleet() error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) theAgentIsListedInFleetAsOnline() error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) systemPackageDashboardsAreListedInFleet() error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) thereIsDataInTheIndex() error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) processStateOnTheHost(process string, state string) error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) agentStateOnTheHost(state string) error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) theAgentIsUnenrolled() error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) theAgentIsNotListedAsOnlineInFleet() error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) thereIsNoDataInTheIndex() error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) anAgentIsEnrolled() error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) theAgentIsReenrolledOnTheHost() error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) isListedInFleetAsOnline(arg1 string) error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) theEnrollmentTokenIsRevoked() error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) anAttemptToEnrollANewAgentFails() error {
	return godog.ErrPending
}
