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

	"github.com/elastic/metricbeat-tests-poc/cli/docker"
	"github.com/elastic/metricbeat-tests-poc/cli/log"
	"github.com/elastic/metricbeat-tests-poc/cli/services"
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
			cleanUpOutputs()
		})
	}, opt)

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

func cleanUpOutputs() {
	dir, _ := os.Getwd()

	files, err := filepath.Glob(dir + "/outputs/*.metrics*")
	log.CheckIfErrorMessage(err, "Cannot remove outputs :(")

	for _, f := range files {
		err := os.Remove(f)
		log.CheckIfErrorMessage(err, "Cannot remove output file "+f+" :(")
	}
}

func metricbeatOutputsMetricsToTheFile(fileName string) error {
	dir, _ := os.Getwd()

	filePath := dir + "/outputs/" + fileName

	foundChannel := make(chan bool, 1)

	go fileChecker(foundChannel)
	initialLoops := 15
	loops := initialLoops
	seconds := 2

	for {
		exists := fileExists(filePath)
		if exists {
			foundChannel <- true
			close(foundChannel)
			break
		}

		if loops == 0 {
			return fmt.Errorf("Could not find the file %s after %d seconds", fileName, (initialLoops * seconds))
		}

		log.Log("Waiting for the file %s to be present (%d seconds left)", fileName, (loops * seconds))
		time.Sleep(time.Duration(seconds) * time.Second)
		loops--
	}

	return nil
}

func fileChecker(c chan bool) {
	if <-c {
		log.Success("File found!")
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) || info.IsDir() {
		return false
	}

	return true
}
