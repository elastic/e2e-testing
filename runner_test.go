package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
)

var opt = godog.Options{Output: colors.Colored(os.Stdout)}

var metricbeatService Service
var serviceManager ServiceManager

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opt)

	serviceManager = NewServiceManager()
}

func TestMain(m *testing.M) {
	flag.Parse()
	opt.Paths = flag.Args()

	status := godog.RunWithOptions("godog", func(s *godog.Suite) {
		s.BeforeScenario(func(interface{}) {
			fmt.Println("Before scenario...")
			cleanUpOutputs()
		})

		s.AfterScenario(func(interface{}, error) {
			fmt.Println("After scenario...")
		})

		ApacheFeatureContext(s)
		MySQLFeatureContext(s)
	}, opt)

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

func cleanUpOutputs() {
	dir, _ := os.Getwd()

	files, err := filepath.Glob(dir + "/outputs/*.metrics")
	if err != nil {
		fmt.Println("Cannot remove outputs :(")
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			fmt.Printf("Cannot remove output file %s :(\n", f)
		}
	}
}

func metricbeatOutputsMetricsToTheFile(fileName string) error {
	time.Sleep(20 * time.Second)

	dir, _ := os.Getwd()

	if _, err := os.Stat(dir + "/outputs/" + fileName); os.IsNotExist(err) {
		return fmt.Errorf("The output file %s does not exist", fileName)
	}

	fmt.Println("Metricbeat outputs to " + fileName)
	return nil
}
