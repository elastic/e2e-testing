package main

import (
	"github.com/DATA-DOG/godog"
	log "github.com/sirupsen/logrus"
)

var mysqlTestSuite MetricbeatTestSuite

func MySQLFeatureContext(s *godog.Suite) {
	s.Step(`^MySQL "([^"]*)" is running for metricbeat "([^"]*)"$`, mySQLIsRunningForMetricbeat)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for MySQL module$`, metricbeatIsInstalledAndConfiguredForMySQLModule)
	s.Step(`^there are no errors in the "([^"]*)" index$`, thereAreNoErrorsInTheIndex)

	s.BeforeScenario(func(interface{}) {
		log.Debug("Before scenario...")
		mysqlTestSuite = MetricbeatTestSuite{}
	})
	s.AfterScenario(func(interface{}, error) {
		log.Debug("After scenario...")
		mysqlTestSuite.CleanUp()
	})
}

func metricbeatIsInstalledAndConfiguredForMySQLModule(metricbeatVersion string) error {
	mysqlService := mysqlTestSuite.Service

	s, err := RunMetricbeatService(metricbeatVersion, mysqlService)
	if err == nil {
		mysqlTestSuite.Metricbeat = s
	}

	query = ElasticsearchQuery{
		EventModule:    "mysql",
		ServiceVersion: mysqlService.GetVersion(),
	}

	return err
}

func mySQLIsRunningForMetricbeat(mysqlVersion string, metricbeatVersion string) error {
	mysqlService := serviceManager.Build("mysql", mysqlVersion, true)

	mysqlService.SetNetworkAlias("mysql_" + mysqlVersion + "-metricbeat_" + metricbeatVersion)

	err := serviceManager.Run(mysqlService)
	if err == nil {
		mysqlTestSuite.Service = mysqlService
	}

	return err
}
