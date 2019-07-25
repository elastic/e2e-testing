package main

import (
	"github.com/DATA-DOG/godog"
)

var mysqlService Service

func MySQLFeatureContext(s *godog.Suite) {
	s.Step(`^MySQL "([^"]*)" is running$`, mySQLIsRunning)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for MySQL module$`, metricbeatIsInstalledAndConfiguredForMySQLModule)
	s.Step(`^metricbeat outputs metrics to the file "([^"]*)"$`, metricbeatOutputsMetricsToTheFile)
}

func metricbeatIsInstalledAndConfiguredForMySQLModule(metricbeatVersion string) error {
	s, err := RunMetricbeatService(metricbeatVersion, mysqlService)

	metricbeatService = s

	return err
}

func mySQLIsRunning(mysqlVersion string) error {
	mysqlService = NewMySQLService(mysqlVersion)

	return serviceManager.Run(mysqlService)
}
