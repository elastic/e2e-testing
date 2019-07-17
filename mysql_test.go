package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"

	testcontainers "github.com/testcontainers/testcontainers-go"
)

var opt = godog.Options{Output: colors.Colored(os.Stdout)}

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opt)
}

func initMySQL(mysqlVersion string) (testcontainers.Container, error) {
	mysqlService := NewMySQLService(mysqlVersion)

	service, err := mysqlService.run()
	if err != nil {
		return nil, err
	}
	defer mysqlService.destroy()

	return service, nil
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
	fmt.Println("Metricbeat " + metricbeatVersion + " is installed")
	return nil
}

func metricbeatOutputsMetricsToTheFile(fileName string) error {
	fmt.Println("Metricbeat outputs to " + fileName)
	return nil
}

func mySQLIsRunning(mysqlVersion string) error {
	container, err := initMySQL(mysqlVersion)

	if err != nil {
		return fmt.Errorf("Could not run MySQL %s: %o", mysqlVersion, err)
	}

	ctx := context.Background()

	ip, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("Could not run MySQL %s: %o", mysqlVersion, err)
	}
	fmt.Printf("MySQL running on IP %s\n", ip)

	port, err := container.MappedPort(ctx, "3306")
	if err != nil {
		return fmt.Errorf("Could not run MySQL %s: %o", mysqlVersion, err)
	}
	fmt.Printf("MySQL running on Port %s\n", port)

	fmt.Println("MySQL " + mysqlVersion + " module works")
	return nil
}

func FeatureContext(s *godog.Suite) {
	s.Step(`^metricbeat "([^"]*)" is installed and configured for MySQL module$`, metricbeatIsInstalledAndConfiguredForMySQLModule)
	s.Step(`^MySQL "([^"]*)" is running$`, mySQLIsRunning)
	s.Step(`^metricbeat outputs metrics to the file "([^"]*)"$`, metricbeatOutputsMetricsToTheFile)

	s.BeforeScenario(func(interface{}) {
		fmt.Println("Before scenario...")
	})
}
