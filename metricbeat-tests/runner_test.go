package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
	log "github.com/sirupsen/logrus"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
	"github.com/elastic/metricbeat-tests-poc/cli/services"
)

var opt = godog.Options{Output: colors.Colored(os.Stdout)}

var metricbeatService services.Service
var query ElasticsearchQuery
var serviceManager services.ServiceManager

// queryMaxAttempts Number of attempts to query elasticsearch before aborting
// It can be overriden by OP_QUERY_MAX_ATTEMPTS env var
var queryMaxAttempts = 5

// queryMetricbeatFetchTimeout Number of seconds that metricbeat has to grab metrics from the module
// It can be overriden by OP_METRICBEAT_FETCH_TIMEOUT env var
var queryMetricbeatFetchTimeout = 20

// queryRetryTimeout Number of seconds between elasticsearch retry queries.
// It can be overriden by OP_RETRY_TIMEOUT env var
var queryRetryTimeout = 3

type ElasticsearchQuery struct {
	EventModule    string
	ServiceVersion string
}

func getIntegerFromEnv(envVar string, defaultValue int) int {
	if value, exists := os.LookupEnv(envVar); exists {
		v, err := strconv.Atoi(value)
		if err == nil {
			return v
		}
	}

	return defaultValue
}

func init() {
	config.Init()

	godog.BindFlags("godog.", flag.CommandLine, &opt)

	serviceManager = services.NewServiceManager()

	queryMaxAttempts = getIntegerFromEnv("OP_QUERY_MAX_ATTEMPTS", queryMaxAttempts)
	queryMetricbeatFetchTimeout = getIntegerFromEnv("OP_METRICBEAT_FETCH_TIMEOUT", queryMetricbeatFetchTimeout)
	queryRetryTimeout = getIntegerFromEnv("OP_RETRY_TIMEOUT", queryRetryTimeout)
}

func TestMain(m *testing.M) {
	flag.Parse()
	opt.Paths = flag.Args()

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

// attempts could be redefined in the OP_QUERY_MAX_ATTEMPTS environment variable
func retrySearch(stackName string, indexName string, esQuery map[string]interface{}, attempts int) (searchResult, error) {
	if attempts == 0 {
		retryMaxTime := queryMaxAttempts * queryRetryTimeout
		err := fmt.Errorf("Could not send query to Elasticsearch in the specified time (%d seconds)", retryMaxTime)

		log.WithFields(log.Fields{
			"error":         err,
			"query":         esQuery,
			"retryAttempts": queryMaxAttempts,
			"retryTimeout":  queryRetryTimeout,
		}).Error(err.Error())

		return searchResult{}, err
	}

	_, err := search("metricbeat", indexName, esQuery)
	if err != nil {
		time.Sleep(time.Duration(queryRetryTimeout) * time.Second)

		log.WithFields(log.Fields{
			"attempt":       attempts,
			"errorCause":    err.Error(),
			"index":         indexName,
			"query":         esQuery,
			"retryAttempts": queryMaxAttempts,
			"retryTimeout":  queryRetryTimeout,
		}).Debugf("Waiting %d seconds for the index to be ready", queryRetryTimeout)

		// recursive approach for retrying the query
		return retrySearch(stackName, indexName, esQuery, (attempts - 1))
	}

	log.WithFields(log.Fields{
		"index":        indexName,
		"query":        esQuery,
		"attempts":     attempts,
		"fetchTimeout": queryMetricbeatFetchTimeout,
	}).Debugf("Waiting %d seconds so that metricbeat is able to grab metrics from the integration module", queryMetricbeatFetchTimeout)
	time.Sleep(time.Duration(queryMetricbeatFetchTimeout) * time.Second)

	return search(stackName, indexName, esQuery)
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

	stackName := "metricbeat"

	result, err := retrySearch(stackName, esIndexName, esQuery, queryMaxAttempts)
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
