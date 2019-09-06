package main

import (
	"github.com/DATA-DOG/godog"
	log "github.com/sirupsen/logrus"
)

var apacheTestSuite MetricbeatTestSuite

func ApacheFeatureContext(s *godog.Suite) {
	s.Step(`^Apache "([^"]*)" is running for metricbeat "([^"]*)"$`, apacheIsRunningForMetricbeat)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for Apache module$`, metricbeatIsInstalledAndConfiguredForApacheModule)
	s.Step(`^there are no errors in the "([^"]*)" index$`, thereAreNoErrorsInTheIndex)

	s.BeforeScenario(func(interface{}) {
		log.Debug("Before scenario...")
		apacheTestSuite = MetricbeatTestSuite{}
	})
	s.AfterScenario(func(interface{}, error) {
		log.Debug("After scenario...")
		apacheTestSuite.CleanUp()
	})
}

func metricbeatIsInstalledAndConfiguredForApacheModule(metricbeatVersion string) error {
	apacheService := apacheTestSuite.Service

	s, err := RunMetricbeatService(metricbeatVersion, apacheService)
	if err == nil {
		apacheTestSuite.Metricbeat = s
	}

	query = ElasticsearchQuery{
		EventModule:    "apache",
		ServiceVersion: apacheService.GetVersion(),
	}

	return err
}

func apacheIsRunningForMetricbeat(apacheVersion string, metricbeatVersion string) error {
	apacheService := serviceManager.Build("apache", apacheVersion, true)

	apacheService.SetNetworkAlias("apache_" + apacheVersion + "-metricbeat_" + metricbeatVersion)

	err := serviceManager.Run(apacheService)
	if err == nil {
		apacheTestSuite.Service = apacheService
	}

	return err
}
