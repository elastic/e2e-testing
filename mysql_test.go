package main

import (
	"github.com/DATA-DOG/godog"
	"github.com/elastic/metricbeat-tests-poc/services"
)

var mysqlService services.Service

func MySQLFeatureContext(s *godog.Suite) {
	s.Step(`^MySQL "([^"]*)" is running$`, mySQLIsRunning)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for MySQL module$`, metricbeatIsInstalledAndConfiguredForMySQLModule)
	s.Step(`^metricbeat outputs metrics to the file "([^"]*)"$`, metricbeatOutputsMetricsToTheFile)
}

func metricbeatIsInstalledAndConfiguredForMySQLModule(metricbeatVersion string) error {
	s, err := services.RunMetricbeatService(metricbeatVersion, mysqlService)

	metricbeatService = s

	return err
}

func mySQLIsRunning(mysqlVersion string) error {
	mysqlService = serviceManager.Build("mysql", mysqlVersion, false)

	return serviceManager.Run(mysqlService)
}
