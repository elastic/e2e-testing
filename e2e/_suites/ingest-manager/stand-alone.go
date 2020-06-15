package main

import "github.com/cucumber/godog"

// StandAloneTestSuite represents the scenarios for Stand-alone-mode
type StandAloneTestSuite struct {
}

func (sats *StandAloneTestSuite) contributeSteps(s *godog.Suite) {
	s.Step(`^a stand-alone agent is deployed$`, sats.aStandaloneAgentIsDeployed)
	s.Step(`^Kibana and Elasticsearch are available$`, sats.kibanaAndElasticsearchAreAvailable)
	s.Step(`^there is new data in the index from agent$`, sats.thereIsNewDataInTheIndexFromAgent)
	s.Step(`^the "([^"]*)" docker container is stopped$`, sats.theDockerContainerIsStopped)
	s.Step(`^there is no new data in the index after agent shuts down$`, sats.thereIsNoNewDataInTheIndexAfterAgentShutsDown)
}

func (sats *StandAloneTestSuite) aStandaloneAgentIsDeployed() error {
	return godog.ErrPending
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
