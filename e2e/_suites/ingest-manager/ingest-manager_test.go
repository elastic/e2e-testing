package main

import (
	"os"
	"path"
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
var stackVersion = "8.0.0-SNAPSHOT"

// profileEnv is the environment to be applied to any execution
// affecting the runtime dependencies (or profile)
var profileEnv map[string]string

// queryRetryTimeout is the number of seconds between elasticsearch retry queries.
// It can be overriden by OP_RETRY_TIMEOUT env var
var queryRetryTimeout = 3

// All URLs running on localhost as Kibana is expected to be exposed there
const kibanaBaseURL = "http://localhost:5601"

func init() {
	config.Init()

	queryRetryTimeout = e2e.GetIntegerFromEnv("OP_RETRY_TIMEOUT", queryRetryTimeout)
	stackVersion = e2e.GetEnv("OP_STACK_VERSION", stackVersion)
}

func IngestManagerFeatureContext(s *godog.Suite) {
	imts := IngestManagerTestSuite{
		Fleet:      &FleetTestSuite{},
		StandAlone: &StandAloneTestSuite{},
	}
	serviceManager := services.NewServiceManager()

	s.Step(`^the "([^"]*)" process is in the "([^"]*)" state on the host$`, imts.processStateOnTheHost)
	s.Step(`^the "([^"]*)" process is "([^"]*)" on the host$`, imts.processStateChangedOnTheHost)

	imts.Fleet.contributeSteps(s)
	imts.StandAlone.contributeSteps(s)

	s.BeforeSuite(func() {
		log.Debug("Installing ingest-manager runtime dependencies")

		workDir, _ := os.Getwd()
		profileEnv = map[string]string{
			"stackVersion":     stackVersion,
			"kibanaConfigPath": path.Join(workDir, "configurations", "kibana.config.yml"),
		}

		profile := "ingest-manager"
		err := serviceManager.RunCompose(true, []string{profile}, profileEnv)
		if err != nil {
			log.WithFields(log.Fields{
				"profile": profile,
			}).Fatal("Could not run the runtime dependencies for the profile.")
		}

		minutesToBeHealthy := 3 * time.Minute
		healthy, err := e2e.WaitForElasticsearch(minutesToBeHealthy)
		if !healthy {
			log.WithFields(log.Fields{
				"error":   err,
				"minutes": minutesToBeHealthy,
			}).Fatal("The Elasticsearch cluster could not get the healthy status")
		}

		healthyKibana, err := e2e.WaitForKibana(minutesToBeHealthy)
		if !healthyKibana {
			log.WithFields(log.Fields{
				"error":   err,
				"minutes": minutesToBeHealthy,
			}).Fatal("The Kibana instance could not get the healthy status")
		}

		imts.Fleet.setup()

		imts.StandAlone.RuntimeDependenciesStartDate = time.Now()
	})
	s.BeforeScenario(func(*messages.Pickle) {
		log.Debug("Before Ingest Manager scenario")

		imts.StandAlone.Cleanup = false
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

		if imts.StandAlone.Cleanup {
			serviceName := "elastic-agent"

			services := []string{serviceName}

			err := serviceManager.RemoveServicesFromCompose("ingest-manager", services, profileEnv)
			if err != nil {
				log.WithFields(log.Fields{
					"service": serviceName,
				}).Error("Could not stop the service.")
			}

			log.WithFields(log.Fields{
				"service": serviceName,
			}).Debug("Service removed from compose.")

			if _, err := os.Stat(imts.StandAlone.AgentConfigFilePath); err == nil {
				os.Remove(imts.StandAlone.AgentConfigFilePath)
				log.WithFields(log.Fields{
					"path": imts.StandAlone.AgentConfigFilePath,
				}).Debug("Elastic Agent configuration file removed.")
			}
		}

		if imts.Fleet.Cleanup {
			serviceName := imts.Fleet.BoxType

			services := []string{serviceName}

			err := serviceManager.RemoveServicesFromCompose("ingest-manager", services, profileEnv)
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
	Fleet      *FleetTestSuite
	StandAlone *StandAloneTestSuite
}

func (imts *IngestManagerTestSuite) processStateChangedOnTheHost(process string, state string) error {
	return godog.ErrPending
}

func (imts *IngestManagerTestSuite) processStateOnTheHost(process string, state string) error {
	// name of the container for the service:
	// we are using the Docker client instead of docker-compose
	// because it does not support returning the output of a
	// command: it simply returns error level
	serviceName := "ingest-manager_elastic-agent_1"
	timeout := 3 * time.Minute

	err := e2e.WaitForProcess(serviceName, process, state, timeout)
	if err != nil {
		if state == "started" {
			log.WithFields(log.Fields{
				"error":   err,
				"timeout": timeout,
			}).Error("The process was not found but should be present")
		} else {
			log.WithFields(log.Fields{
				"error":   err,
				"timeout": timeout,
			}).Error("The process was found but shouldn't be present")
		}

		return err
	}

	return nil
}
