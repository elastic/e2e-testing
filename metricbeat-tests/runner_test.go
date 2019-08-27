package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
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
var query ElasticsearchQuery
var serviceManager services.ServiceManager

type ElasticsearchQuery struct {
	EventModule    string
	ServiceVersion string
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

// assertHitsArePresent returns an error if no hits are present
func assertHitsArePresent(hits map[string]interface{}, q ElasticsearchQuery) error {
	hitsCount := len(hits["hits"].(map[string]interface{})["hits"].([]interface{}))
	if hitsCount == 0 {
		return fmt.Errorf(
			"There aren't documents for %s-%s on Metricbeat index",
			q.EventModule, q.ServiceVersion)
	}

	return nil
}

// assertHitsDoNotContainErrors returns an error if any of the returned entries contains
// an "error.message" field in the "_source" document
func assertHitsDoNotContainErrors(hits map[string]interface{}, q ElasticsearchQuery) error {
	for _, hit := range hits["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		if val, ok := source.(map[string]interface{})["error"]; ok {
			if msg, exists := val.(map[string]interface{})["message"]; exists {
				log.WithFields(log.Fields{
					"ID":            hit.(map[string]interface{})["_id"],
					"error.message": msg,
				}).Error("Error Hit found")

				return fmt.Errorf(
					"There are errors for %s-%s on Metricbeat index",
					q.EventModule, q.ServiceVersion)
			}
		}
	}

	return nil
}

func thereAreNoErrorsInTheIndex(index string) error {
	esIndexName := strings.ReplaceAll(index, "-SNAPSHOT", "")
	now := time.Now()

	formattedDate := strings.ReplaceAll(now.Format("2006-01-02"), "-", ".")

	// TODO: this index name is hardcoded after checking the index name on Kibana
	// I would need help setting up the index name from metricbeat configuration
	esIndexName += "-" + formattedDate + "-000001"

	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"event.module": query.EventModule,
						},
					},
					{
						"match": map[string]interface{}{
							"service.version": query.ServiceVersion,
						},
					},
				},
			},
		},
	}

	result, err := search("metricbeat", esIndexName, esQuery)
	if err != nil {
		return err
	}

	r := result.Result

	err = assertHitsArePresent(r, query)
	if err != nil {
		return err
	}

	err = assertHitsDoNotContainErrors(r, query)
	if err != nil {
		return err
	}

	return nil
}
