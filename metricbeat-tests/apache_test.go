package main

import (
	"github.com/DATA-DOG/godog"
	"github.com/elastic/metricbeat-tests-poc/cli/services"
)

var apacheService services.Service

func ApacheFeatureContext(s *godog.Suite) {
	s.Step(`^Apache "([^"]*)" is running for metricbeat "([^"]*)"$`, apacheIsRunningForMetricbeat)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for Apache module$`, metricbeatIsInstalledAndConfiguredForApacheModule)
	s.Step(`^there are no errors in the "([^"]*)" index$`, thereAreNoErrorsInTheIndex)
}

func metricbeatIsInstalledAndConfiguredForApacheModule(metricbeatVersion string) error {
	s, err := RunMetricbeatService(metricbeatVersion, apacheService)

	metricbeatService = s

	query = ElasticsearchQuery{
		EventModule:    "apache",
		ServiceVersion: apacheService.GetVersion(),
	}

	return err
}

func apacheIsRunningForMetricbeat(apacheVersion string, metricbeatVersion string) error {
	apacheService = serviceManager.Build("apache", apacheVersion, false)

	apacheService.SetNetworkAlias("apache_" + apacheVersion + "-metricbeat_" + metricbeatVersion)

	return serviceManager.Run(apacheService)
}
