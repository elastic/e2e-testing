package main

import (
	"context"
	"fmt"

	"github.com/DATA-DOG/godog"
)

var mysqlService Service

func MySQLFeatureContext(s *godog.Suite) {
	s.Step(`^MySQL "([^"]*)" is running$`, mySQLIsRunning)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for MySQL module$`, metricbeatIsInstalledAndConfiguredForMySQLModule)
	s.Step(`^metricbeat outputs metrics to the file "([^"]*)"$`, metricbeatOutputsMetricsToTheFile)
}

func metricbeatIsInstalledAndConfiguredForMySQLModule(metricbeatVersion string) error {
	metricbeatService, err := NewMetricbeatService(metricbeatVersion, mysqlService)
	if err != nil {
		return err
	}
	if metricbeatService == nil {
		return fmt.Errorf("Could not create Metricbeat %s service for MySQL", metricbeatVersion)
	}

	container, err := metricbeatService.Run()
	if err != nil || container == nil {
		return fmt.Errorf("Could not run Metricbeat %s: %v", metricbeatVersion, err)
	}

	ctx := context.Background()

	ip, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("Could not run Metricbeat %s: %v", metricbeatVersion, err)
	}
	fmt.Printf("Metricbeat %s is running configured for MySQL on IP %s\n", metricbeatVersion, ip)

	return nil
}

func mySQLIsRunning(mysqlVersion string) error {
	mysqlService = NewMySQLService(mysqlVersion)

	return serviceManager.Run(mysqlService)
}
