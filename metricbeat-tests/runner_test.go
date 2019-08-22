package main

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
	log "github.com/sirupsen/logrus"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
	"github.com/elastic/metricbeat-tests-poc/cli/docker"
	"github.com/elastic/metricbeat-tests-poc/cli/services"
)

var opt = godog.Options{Output: colors.Colored(os.Stdout)}

var metricbeatService services.Service
var query ElasticsearchQuery
var serviceManager services.ServiceManager

type ElasticsearchQuery struct {
	EventDataset string
}

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
		})

		s.AfterScenario(func(interface{}, error) {
			log.Debug("After scenario...")
		})
	}, opt)

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

func metricbeatStoresMetricsToElasticsearchInTheIndex(metricbeatVersion string) error {
	esIndexName := strings.ReplaceAll(metricbeatVersion, "-SNAPSHOT", "")

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}

	result, err := search(esIndexName, query)
	if err != nil {
		return err
	}

	r := result.Result

	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		log.WithFields(log.Fields{
			"ID":     hit.(map[string]interface{})["_id"],
			"source": hit.(map[string]interface{})["_source"],
		}).Info("Hit found")
	}

	return nil
}
