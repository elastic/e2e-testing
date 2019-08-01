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
	"github.com/elastic/metricbeat-tests-poc/docker"
	"github.com/elastic/metricbeat-tests-poc/log"
	"github.com/elastic/metricbeat-tests-poc/services"
)

var opt = godog.Options{Output: colors.Colored(os.Stdout)}

var metricbeatService services.Service
var serviceManager services.ServiceManager

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opt)

	serviceManager = services.NewServiceManager()
}

func TestMain(m *testing.M) {
	flag.Parse()
	opt.Paths = flag.Args()

	docker.GetDevNetwork()

	status := godog.RunWithOptions("godog", func(s *godog.Suite) {
		s.BeforeScenario(func(interface{}) {
			log.Info("Before scenario...")
			cleanUpOutputs()
		})

		s.AfterScenario(func(interface{}, error) {
			log.Info("After scenario...")
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
	log.CheckIfErrorMessage(err, "Cannot remove outputs :(")

	for _, f := range files {
		err := os.Remove(f)
		log.CheckIfErrorMessage(err, "Cannot remove output file "+f+" :(")
	}
}

func metricbeatOutputsMetricsToTheFile(fileName string) error {
	time.Sleep(20 * time.Second)

	dir, _ := os.Getwd()

	if _, err := os.Stat(dir + "/outputs/" + fileName); os.IsNotExist(err) {
		return fmt.Errorf("The output file %s does not exist", fileName)
	}

	log.Info("Metricbeat outputs to " + fileName)
	return nil
}
