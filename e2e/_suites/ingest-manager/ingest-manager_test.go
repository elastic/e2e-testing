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
var stackVersion = "7.7.0"

// profileEnv is the environment to be applied to any execution
// affecting the runtime dependencies (or profile)
var profileEnv map[string]string

// All URLs running on localhost as Kibana is expected to be exposed there
const kibanaBaseURL = "http://localhost:5601"

func init() {
	config.Init()

	stackVersion = e2e.GetEnv("OP_STACK_VERSION", stackVersion)
}

func IngestManagerFeatureContext(s *godog.Suite) {
	imts := IngestManagerTestSuite{
		Fleet:      &FleetTestSuite{},
		StandAlone: &StandAloneTestSuite{},
	}
	serviceManager := services.NewServiceManager()

	s.Step(`^the "([^"]*)" process is "([^"]*)" on the host$`, imts.processStateOnTheHost)

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
		}
	})
}

// IngestManagerTestSuite represents a test suite, holding references to the pieces needed to run the tests
type IngestManagerTestSuite struct {
	Fleet      *FleetTestSuite
	StandAlone *StandAloneTestSuite
}

func (imts *IngestManagerTestSuite) processStateOnTheHost(process string, state string) error {
	log.WithFields(log.Fields{
		"process": process,
		"state":   state,
	}).Debug("Checking process state on the host")

	return godog.ErrPending
}
