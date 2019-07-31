package main

import (
	"github.com/DATA-DOG/godog"
	"github.com/elastic/metricbeat-tests-poc/services"
)

var apacheService services.Service

func ApacheFeatureContext(s *godog.Suite) {
	s.Step(`^Apache "([^"]*)" is running$`, apacheIsRunning)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for Apache module$`, metricbeatIsInstalledAndConfiguredForApacheModule)
	s.Step(`^metricbeat outputs metrics to the file "([^"]*)"$`, metricbeatOutputsMetricsToTheFile)
}

func metricbeatIsInstalledAndConfiguredForApacheModule(metricbeatVersion string) error {
	s, err := services.RunMetricbeatService(metricbeatVersion, apacheService)

	metricbeatService = s

	return err
}

func apacheIsRunning(apacheVersion string) error {
	apacheService = serviceManager.Build("apache", apacheVersion, false)

	return serviceManager.Run(apacheService)
}
