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
	log "github.com/sirupsen/logrus"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
	"github.com/elastic/metricbeat-tests-poc/cli/docker"
	"github.com/elastic/metricbeat-tests-poc/cli/services"
)

var opt = godog.Options{Output: colors.Colored(os.Stdout)}

var metricbeatService services.Service
var serviceManager services.ServiceManager

func init() {
	config.InitConfig()

	godog.BindFlags("godog.", flag.CommandLine, &opt)

	serviceManager = services.NewServiceManager()
}

func TestMain(m *testing.M) {
	flag.Parse()
	opt.Paths = flag.Args()

	docker.GetDevNetwork()

	status := godog.RunWithOptions("godog", func(s *godog.Suite) {
		s.BeforeScenario(func(interface{}) {
			log.Debug("Before scenario...")
			cleanUpOutputs()
		})

		s.AfterScenario(func(interface{}, error) {
			log.Debug("After scenario...")
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

	pattern := dir + "/outputs/*.metrics*"

	files, err := filepath.Glob(pattern)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"path":  pattern,
		}).Fatal("Cannot retrieve outputs :(")
	}

	for _, f := range files {
		err := os.Remove(f)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"path":  f,
			}).Fatal("Cannot remove output file :(")
		}
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

		log.WithFields(log.Fields{
			"file":    fileName,
			"times":   loops,
			"seconds": (loops * seconds),
		}).Debug("Waiting for the file to be present")

		time.Sleep(time.Duration(seconds) * time.Second)
		loops--
	}

	return nil
}

func fileChecker(c chan bool) {
	if <-c {
		log.Info("File found!")
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) || info.IsDir() {
		return false
	}

	return true
}
