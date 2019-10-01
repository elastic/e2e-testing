package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
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

var query ElasticsearchQuery
var serviceManager services.ServiceManager

// queryMaxAttempts is the number of attempts to query elasticsearch before aborting
// It can be overriden by OP_QUERY_MAX_ATTEMPTS env var
var queryMaxAttempts = 5

// queryMetricbeatFetchTimeout is the number of seconds that metricbeat has to grab metrics from the module
// It can be overriden by OP_METRICBEAT_FETCH_TIMEOUT env var
var queryMetricbeatFetchTimeout = 20

// queryRetryTimeout is the number of seconds between elasticsearch retry queries.
// It can be overriden by OP_RETRY_TIMEOUT env var
var queryRetryTimeout = 3

// MetricbeatTestSuite represents a test suite, holding references to both metricbeat ant
// the service to be monitored
type MetricbeatTestSuite struct {
	IndexName      string // the unique name for the index to be used in this test suite
	ServiceName    string // the service to be monitored by metricbeat
	ServiceVersion string // the version of the service to be monitored by metricbeat
	Version        string // the metricbeat version for the test
}

// As we are using an index per scenario outline, with an index name formed by metricbeat-version1-module-version2,
// and because of the ILM is configured on metricbeat side, then we can use an asterisk for the index name:
// each scenario outline will be namespaced, so no collitions between different test cases should appear
func (mts *MetricbeatTestSuite) setIndexName() {
	mVersion := strings.ReplaceAll(mts.Version, "-SNAPSHOT", "")

	index := fmt.Sprintf("metricbeat-%s-%s-%s", mVersion, mts.ServiceName, mts.ServiceVersion)

	index += "-" + strings.ToLower(randomString(8))

	mts.IndexName = index
}

// CleanUp cleans up services in the test suite
func (mts *MetricbeatTestSuite) CleanUp() error {
	serviceManager := services.NewServiceManager()

	fn := func(ctx context.Context) error {
		return deleteIndex(ctx, "metricbeat", mts.IndexName)
	}
	defer fn(context.Background())

	err := serviceManager.RemoveServicesFromCompose(
		"metricbeat", []string{"metricbeat", mts.ServiceName})
	if err != nil {
		log.WithFields(log.Fields{
			"service": mts.ServiceName,
		}).Error("Could not stop the service.")
	}

	log.WithFields(log.Fields{
		"service": mts.ServiceName,
	}).Debug("Service removed from compose.")

	return err
}

func (mts *MetricbeatTestSuite) installedAndConfiguredForModule(version string, serviceType string) error {
	serviceType = strings.ToLower(serviceType)

	// at this point we have everything to define the index name
	mts.Version = version
	mts.setIndexName()

	err := mts.runMetricbeatService()
	if err != nil {
		return err
	}

	query = ElasticsearchQuery{
		EventModule:    serviceType,
		ServiceVersion: mts.ServiceVersion,
	}

	return nil
}

// runMetricbeatService runs a metricbeat service entity for a service to monitor it
func (mts *MetricbeatTestSuite) runMetricbeatService() error {
	dir, _ := os.Getwd()

	serviceManager := services.NewServiceManager()

	env := map[string]string{
		"BEAT_STRICT_PERMS":     "false",
		"indexName":             mts.IndexName,
		"metricbeatConfigFile":  path.Join(dir, "configurations", mts.ServiceName+".yml"),
		"metricbeatTag":         mts.Version,
		mts.ServiceName + "Tag": mts.ServiceVersion,
		"serviceName":           mts.ServiceName,
	}

	err := serviceManager.AddServicesToCompose("metricbeat", []string{"metricbeat"}, env)
	if err != nil {
		msg := fmt.Sprintf("Could not run Metricbeat %s for %s %v", mts.Version, mts.ServiceName, err)

		log.WithFields(log.Fields{
			"error":             err,
			"metricbeatVersion": mts.Version,
			"service":           mts.ServiceName,
			"serviceVersion":    mts.ServiceVersion,
		}).Error(msg)

		return err
	}

	log.WithFields(log.Fields{
		"metricbeatVersion": mts.Version,
		"service":           mts.ServiceName,
		"serviceVersion":    mts.ServiceVersion,
	}).Info("Metricbeat is running configured for the service")

	return nil
}

func (mts *MetricbeatTestSuite) serviceIsRunningForMetricbeat(
	serviceType string, serviceVersion string, metricbeatVersion string) error {

	serviceType = strings.ToLower(serviceType)

	env := map[string]string{
		serviceType + "Tag": serviceVersion,
	}

	err := serviceManager.AddServicesToCompose("metricbeat", []string{serviceType}, env)
	if err != nil {
		log.WithFields(log.Fields{
			"service": serviceType,
			"version": serviceVersion,
		}).Error("Could not run the service.")
	}

	mts.ServiceName = serviceType
	mts.ServiceVersion = serviceVersion

	return err
}

func (mts *MetricbeatTestSuite) thereAreNoErrorsInTheIndex() error {
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"event.module": query.EventModule,
						},
					},
				},
			},
		},
	}

	stackName := "metricbeat"

	_, err := retrySearch(stackName, mts.IndexName, esQuery, queryMaxAttempts)
	if err != nil {
		return err
	}

	time.Sleep(time.Duration(queryMetricbeatFetchTimeout) * time.Second)
	result, err := search(stackName, mts.IndexName, esQuery)
	if err != nil {
		return err
	}

	err = assertHitsArePresent(result, query)
	if err != nil {
		return err
	}

	err = assertHitsDoNotContainErrors(result, query)
	if err != nil {
		return err
	}

	return nil
}

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

	status := godog.RunWithOptions("godog", func(s *godog.Suite) {}, opt)

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

// attempts could be redefined in the OP_QUERY_MAX_ATTEMPTS environment variable
func retrySearch(stackName string, indexName string, esQuery map[string]interface{}, attempts int) (searchResult, error) {
	totalRetryTime := attempts * queryRetryTimeout

	for attempts := attempts; attempts > 0; attempts-- {
		result, err := search(stackName, indexName, esQuery)
		if err == nil {
			return result, nil
		}

		log.WithFields(log.Fields{
			"attempt":       attempts,
			"errorCause":    err.Error(),
			"index":         indexName,
			"query":         esQuery,
			"retryAttempts": queryMaxAttempts,
			"retryTimeout":  queryRetryTimeout,
		}).Warnf("Waiting %d seconds for the index to be ready", queryRetryTimeout)
		if attempts > 1 {
			time.Sleep(time.Duration(queryRetryTimeout) * time.Second)
		}
	}

	err := fmt.Errorf("Could not send query to Elasticsearch in the specified time (%d seconds)", totalRetryTime)

	log.WithFields(log.Fields{
		"error":         err,
		"query":         esQuery,
		"retryAttempts": queryMaxAttempts,
		"retryTimeout":  queryRetryTimeout,
	}).Error(err.Error())

	return searchResult{}, err
}
