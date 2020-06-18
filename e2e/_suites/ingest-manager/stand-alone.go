package main

import (
	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/cli/services"
	"github.com/elastic/e2e-testing/e2e"
	log "github.com/sirupsen/logrus"
)

// StandAloneTestSuite represents the scenarios for Stand-alone-mode
type StandAloneTestSuite struct {
	AgentConfigFilePath string
	Cleanup             bool
}

func (sats *StandAloneTestSuite) contributeSteps(s *godog.Suite) {
	s.Step(`^a stand-alone agent is deployed$`, sats.aStandaloneAgentIsDeployed)
	s.Step(`^Kibana and Elasticsearch are available$`, sats.kibanaAndElasticsearchAreAvailable)
	s.Step(`^there is new data in the index from agent$`, sats.thereIsNewDataInTheIndexFromAgent)
	s.Step(`^the "([^"]*)" docker container is stopped$`, sats.theDockerContainerIsStopped)
	s.Step(`^there is no new data in the index after agent shuts down$`, sats.thereIsNoNewDataInTheIndexAfterAgentShutsDown)
}

func (sats *StandAloneTestSuite) aStandaloneAgentIsDeployed() error {
	log.Debug("Deploying an agent to Fleet")

	serviceManager := services.NewServiceManager()

	profile := "ingest-manager"
	serviceName := "elastic-agent"

	configurationFileURL := "https://raw.githubusercontent.com/elastic/beats/master/x-pack/elastic-agent/elastic-agent.docker.yml"

	configurationFilePath, err := e2e.DownloadFile(configurationFileURL)
	if err != nil {
		return err
	}
	sats.AgentConfigFilePath = configurationFilePath

	profileEnv["elasticAgentConfigFile"] = sats.AgentConfigFilePath

	err = serviceManager.AddServicesToCompose(profile, []string{serviceName}, profileEnv)
	if err != nil {
		log.Error("Could not deploy the elastic-agent")
		return err
	}

	sats.Cleanup = true

	if log.IsLevelEnabled(log.DebugLevel) {
		composes := []string{
			profile,     // profile name
			serviceName, // agent service
		}
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

func (sats *StandAloneTestSuite) kibanaAndElasticsearchAreAvailable() error {
	return godog.ErrPending
}

func (sats *StandAloneTestSuite) thereIsNewDataInTheIndexFromAgent() error {
	return godog.ErrPending
}

func (sats *StandAloneTestSuite) theDockerContainerIsStopped(arg1 string) error {
	return godog.ErrPending
}

func (sats *StandAloneTestSuite) thereIsNoNewDataInTheIndexAfterAgentShutsDown() error {
	return godog.ErrPending
}
