package main

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
)

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

func anElasticStackInVersionIsRunning(esVersion string) error {
	fmt.Println("Elastic stack " + esVersion + " is running")
	return nil
}

func metricbeatIsInstalledAndConfiguredForMySQLModule(metricbeatVersion string) error {
	fmt.Println("Metricbeat " + metricbeatVersion + " is installed")
	return nil
}

func iWantToCheckThatItsWorkingForMySQL(mysqlVersion string) error {
	fmt.Println("MySQL " + mysqlVersion + " module works")
	return nil
}

func FeatureContext(s *godog.Suite) {
	s.Step(`^an Elastic stack in version "([^"]*)" is running$`, anElasticStackInVersionIsRunning)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for MySQL module$`, metricbeatIsInstalledAndConfiguredForMySQLModule)
	s.Step(`^I want to check that it\'s working for MySQL "([^"]*)"$`, iWantToCheckThatItsWorkingForMySQL)

	s.BeforeScenario(func(interface{}) {
		fmt.Println("Before scenario...")
	})
}
