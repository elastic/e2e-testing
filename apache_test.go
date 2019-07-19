package main

import (
	"context"
	"fmt"

	"github.com/DATA-DOG/godog"
)

var apacheService Service

func ApacheFeatureContext(s *godog.Suite) {
	s.Step(`^Apache "([^"]*)" is running on port "([^"]*)"$`, apacheIsRunningOnPort)
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

func apacheIsRunningOnPort(apacheVersion string, port string) error {
	apacheService = NewApacheService(apacheVersion, port)

	container, err := apacheService.Run()
	if err != nil {
		return fmt.Errorf("Could not run Apache %s: %v", apacheVersion, err)
	}

	ctx := context.Background()

	ip, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("Could not run Apache %s: %v", apacheVersion, err)
	}

	fmt.Printf("Apache %s is running on %s:%s\n", apacheVersion, ip, port)

	return nil
}
