package main

import (
	"context"
	"fmt"

	"github.com/DATA-DOG/godog"
)

var apacheService Service

func ApacheFeatureContext(s *godog.Suite) {
	s.Step(`^Apache "([^"]*)" is running$`, apacheIsRunning)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for Apache module$`, metricbeatIsInstalledAndConfiguredForApacheModule)
	s.Step(`^metricbeat outputs metrics to the file "([^"]*)"$`, metricbeatOutputsMetricsToTheFile)
}

func metricbeatIsInstalledAndConfiguredForApacheModule(metricbeatVersion string) error {
	metricbeatService, err := NewMetricbeatService(metricbeatVersion, apacheService)
	if err != nil {
		return err
	}
	if metricbeatService == nil {
		return fmt.Errorf("Could not create Metricbeat %s service for Apache", metricbeatVersion)
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
	fmt.Printf("Metricbeat %s is running configured for Apache on IP %s\n", metricbeatVersion, ip)

	return nil
}

func apacheIsRunning(apacheVersion string) error {
	apacheService = NewApacheService(apacheVersion)

	return serviceManager.Run(apacheService)
}
