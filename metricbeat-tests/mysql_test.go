package main

import (
	"github.com/DATA-DOG/godog"
	"github.com/elastic/metricbeat-tests-poc/cli/services"
)

var mysqlService services.Service

func MySQLFeatureContext(s *godog.Suite) {
	s.Step(`^MySQL "([^"]*)" is running$`, mySQLIsRunning)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for MySQL module$`, metricbeatIsInstalledAndConfiguredForMySQLModule)
	s.Step(`^there are no errors in the "([^"]*)" index$`, thereAreNoErrorsInTheIndex)
}

func metricbeatIsInstalledAndConfiguredForMySQLModule(metricbeatVersion string) error {
	s, err := RunMetricbeatService(metricbeatVersion, mysqlService)

	metricbeatService = s

	query = ElasticsearchQuery{
		EventModule: "mysql",
	}

	return err
}

func mySQLIsRunning(mysqlVersion string) error {
	mysqlService = serviceManager.Build("mysql", mysqlVersion, false)

	return serviceManager.Run(mysqlService)
}
