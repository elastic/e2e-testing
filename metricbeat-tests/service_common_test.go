package main

import (
	"github.com/DATA-DOG/godog"
	log "github.com/sirupsen/logrus"
)

func FeatureContext(s *godog.Suite) {
	testSuite := MetricbeatTestSuite{}

	s.Step(`^([^"]*) "([^"]*)" is running for metricbeat "([^"]*)"$`, testSuite.serviceIsRunningForMetricbeat)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for ([^"]*) module$`, testSuite.installedAndConfiguredForModule)
	s.Step(`^there are no errors in the index$`, testSuite.thereAreNoErrorsInTheIndex)

	s.BeforeScenario(func(interface{}) {
		log.Debug("Before scenario...")
	})
	s.AfterScenario(func(interface{}, error) {
		log.Debug("After scenario...")
		testSuite.CleanUp()
	})
}
