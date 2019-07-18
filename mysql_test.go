package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
)

var metricbeatService Service
var mysqlService Service

var opt = godog.Options{Output: colors.Colored(os.Stdout)}

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opt)
}

func TestMain(m *testing.M) {
	flag.Parse()
	opt.Paths = flag.Args()

	status := godog.RunWithOptions("MySQL", func(s *godog.Suite) {
		FeatureContext(s)
	}, opt)

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

func metricbeatIsInstalledAndConfiguredForMySQLModule(metricbeatVersion string) error {
	metricbeatService = NewMetricbeatService(metricbeatVersion, mysqlService)

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

func metricbeatOutputsMetricsToTheFile(fileName string) error {
	fmt.Println("Metricbeat outputs to " + fileName)
	return nil
}

func mySQLIsRunning(mysqlVersion string) error {
	mysqlService = NewMySQLService(mysqlVersion)

	container, err := mysqlService.Run()
	if err != nil {
		return fmt.Errorf("Could not run MySQL %s: %o", mysqlVersion, err)
	}

	ctx := context.Background()

	ip, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("Could not run MySQL %s: %s", mysqlVersion, err)
	}
	port, err := container.MappedPort(ctx, "3306")
	if err != nil {
		return fmt.Errorf("Could not run MySQL %s: %s", mysqlVersion, err)
	}

	fmt.Printf("MySQL %s is running on %s:%s\n", mysqlVersion, ip, port)

	return nil
}

func FeatureContext(s *godog.Suite) {
	s.Step(`^MySQL "([^"]*)" is running$`, mySQLIsRunning)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for MySQL module$`, metricbeatIsInstalledAndConfiguredForMySQLModule)
	s.Step(`^metricbeat outputs metrics to the file "([^"]*)"$`, metricbeatOutputsMetricsToTheFile)

	s.BeforeScenario(func(interface{}) {
		fmt.Println("Before scenario...")
	})

	s.AfterScenario(func(scenario interface{}, err error) {
		seconds := 100
		pause := time.Duration(seconds) * time.Second

		fmt.Printf("Sleeping %d seconds after scenario...\n", seconds)

		time.Sleep(pause)
	})
}
