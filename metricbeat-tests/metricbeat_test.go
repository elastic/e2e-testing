package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/DATA-DOG/godog"
	"github.com/elastic/metricbeat-tests-poc/cli/config"
	"github.com/elastic/metricbeat-tests-poc/cli/services"
	log "github.com/sirupsen/logrus"
)

// metricbeatVersion is the version of the metricbeat to use
// It can be overriden by OP_METRICBEAT_VERSION env var
var metricbeatVersion = "7.6.0"

//nolint:unused
var query ElasticsearchQuery

var serviceManager services.ServiceManager

func init() {
	metricbeatVersion = getEnv("OP_METRICBEAT_VERSION", metricbeatVersion)
	serviceManager = services.NewServiceManager()
}

// MetricbeatTestSuite represents a test suite, holding references to both metricbeat ant
// the service to be monitored
//nolint:unused
type MetricbeatTestSuite struct {
	cleanUpTmpFiles   bool   // if it's needed to clean up temporary files
	configurationFile string // the  name of the configuration file to be used in this test suite
	IndexName         string // the unique name for the index to be used in this test suite
	ServiceName       string // the service to be monitored by metricbeat
	ServiceType       string // the type of the service to be monitored by metricbeat
	ServiceVariant    string // the variant of the service to be monitored by metricbeat
	ServiceVersion    string // the version of the service to be monitored by metricbeat
	Version           string // the metricbeat version for the test
}

// As we are using an index per scenario outline, with an index name formed by metricbeat-version1-module-version2,
// and because of the ILM is configured on metricbeat side, then we can use an asterisk for the index name:
// each scenario outline will be namespaced, so no collitions between different test cases should appear
func (mts *MetricbeatTestSuite) setIndexName() {
	mVersion := strings.ReplaceAll(mts.Version, "-SNAPSHOT", "")

	var index string
	if mts.ServiceName != "" {
		index = fmt.Sprintf("metricbeat-%s-%s-%s", mVersion, mts.ServiceName, mts.ServiceVersion)
	} else {
		index = fmt.Sprintf("metricbeat-%s", mVersion)
	}

	index += "-" + strings.ToLower(randomString(8))

	mts.IndexName = index
}

// CleanUp cleans up services in the test suite
func (mts *MetricbeatTestSuite) CleanUp() error {
	serviceManager := services.NewServiceManager()

	fn := func(ctx context.Context) {
		err := deleteIndex(ctx, "metricbeat", mts.IndexName)
		if err != nil {
			log.WithFields(log.Fields{
				"stack": "metricbeat",
				"index": mts.IndexName,
			}).Warn("The index was not deleted, but we are not failing the test case")
		}
	}
	defer fn(context.Background())

	env := map[string]string{
		"stackVersion": stackVersion,
	}

	services := []string{"metricbeat"}
	if mts.ServiceName != "" {
		services = append(services, mts.ServiceName)
	}

	err := serviceManager.RemoveServicesFromCompose("metricbeat", services, env)
	if err != nil {
		log.WithFields(log.Fields{
			"service": mts.ServiceName,
		}).Error("Could not stop the service.")
	}

	log.WithFields(log.Fields{
		"service": mts.ServiceName,
	}).Debug("Service removed from compose.")

	if mts.cleanUpTmpFiles {
		if _, err := os.Stat(mts.configurationFile); err == nil {
			os.Remove(mts.configurationFile)
			log.WithFields(log.Fields{
				"path": mts.configurationFile,
			}).Debug("Metricbeat configuration file removed.")
		}
	}

	return err
}

// MetricbeatFeatureContext adds steps to the Godog test suite
//nolint:deadcode,unused
func MetricbeatFeatureContext(s *godog.Suite) {
	testSuite := MetricbeatTestSuite{}

	s.Step(`^([^"]*) "([^"]*)" is running for metricbeat$`, testSuite.serviceIsRunningForMetricbeat)
	s.Step(`^"([^"]*)" v([^"]*), variant of "([^"]*)", is running for metricbeat$`, testSuite.serviceVariantIsRunningForMetricbeat)
	s.Step(`^metricbeat is installed and configured for ([^"]*) module$`, testSuite.installedAndConfiguredForModule)
	s.Step(`^metricbeat is installed and configured for "([^"]*)", variant of the "([^"]*)" module$`, testSuite.installedAndConfiguredForVariantModule)
	s.Step(`^metricbeat waits "([^"]*)" seconds for the service$`, testSuite.waitsSeconds)
	s.Step(`^metricbeat runs for "([^"]*)" seconds$`, testSuite.runsForSeconds)
	s.Step(`^there are no errors in the index$`, testSuite.thereAreNoErrorsInTheIndex)
	s.Step(`^there are "([^"]*)" events in the index$`, testSuite.thereAreEventsInTheIndex)

	s.Step(`^metricbeat is installed using "([^"]*)" configuration$`, testSuite.installedUsingConfiguration)

	s.BeforeScenario(func(interface{}) {
		log.Debug("Before scenario...")
	})
	s.AfterScenario(func(interface{}, error) {
		log.Debug("After scenario...")
		err := testSuite.CleanUp()
		if err != nil {
			log.Errorf("CleanUp failed: %v", err)
		}
	})
}

func (mts *MetricbeatTestSuite) installedAndConfiguredForModule(serviceType string) error {
	serviceType = strings.ToLower(serviceType)

	// at this point we have everything to define the index name
	mts.Version = metricbeatVersion
	mts.setIndexName()
	mts.ServiceType = serviceType

	// look up configurations under workspace's configurations directory
	dir, _ := os.Getwd()
	mts.configurationFile = path.Join(dir, "configurations", mts.ServiceName+".yml")

	return nil
}

func (mts *MetricbeatTestSuite) installedAndConfiguredForVariantModule(serviceVariant string, serviceType string) error {
	serviceType = strings.ToLower(serviceType)

	// at this point we have everything to define the index name
	mts.Version = metricbeatVersion
	mts.setIndexName()
	mts.ServiceVariant = serviceVariant
	mts.ServiceType = serviceType

	// look up configurations under workspace's configurations directory
	dir, _ := os.Getwd()
	mts.configurationFile = path.Join(dir, "configurations", mts.ServiceName+".yml")

	return nil
}

func (mts *MetricbeatTestSuite) installedUsingConfiguration(configuration string) error {
	// at this point we have everything to define the index name
	mts.Version = metricbeatVersion
	mts.setIndexName()

	// use master branch for snapshots
	tag := "v" + metricbeatVersion
	if strings.Contains(metricbeatVersion, "SNAPSHOT") {
		tag = "master"
	}

	configurationFileURL := "https://raw.githubusercontent.com/elastic/beats/" + tag + "/metricbeat/" + configuration + ".yml"

	configurationFilePath, err := downloadFile(configurationFileURL)
	if err != nil {
		return err
	}
	mts.configurationFile = configurationFilePath
	mts.cleanUpTmpFiles = true

	err = mts.runMetricbeatService()
	if err != nil {
		return err
	}

	query = ElasticsearchQuery{
		EventModule:    "system",
		ServiceVersion: mts.Version,
	}

	return nil
}

// runsForSeconds waits for a number of seconds so that metricbeat gets
// an acceptable number of metrics
func (mts *MetricbeatTestSuite) runsForSeconds(seconds string) error {
	err := mts.runMetricbeatService()
	if err != nil {
		return err
	}

	query = ElasticsearchQuery{
		EventModule:    mts.ServiceType,
		ServiceVersion: mts.ServiceVersion,
	}

	return sleep(seconds)
}

// runMetricbeatService runs a metricbeat service entity for a service to monitor it
func (mts *MetricbeatTestSuite) runMetricbeatService() error {
	serviceManager := services.NewServiceManager()

	env := map[string]string{
		"BEAT_STRICT_PERMS":     "false",
		"indexName":             mts.IndexName,
		"metricbeatConfigFile":  mts.configurationFile,
		"metricbeatTag":         mts.Version,
		"stackVersion":          stackVersion,
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

func (mts *MetricbeatTestSuite) serviceIsRunningForMetricbeat(serviceType string, serviceVersion string) error {
	serviceType = strings.ToLower(serviceType)

	env := map[string]string{
		"stackVersion": stackVersion,
	}
	env = config.PutServiceEnvironment(env, serviceType, serviceVersion)

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

func (mts *MetricbeatTestSuite) serviceVariantIsRunningForMetricbeat(
	serviceVariant string, serviceVersion string, serviceType string) error {

	serviceVariant = strings.ToLower(serviceVariant)
	serviceType = strings.ToLower(serviceType)

	env := map[string]string{
		"stackVersion": stackVersion,
	}
	env = config.PutServiceVariantEnvironment(env, serviceType, serviceVariant, serviceVersion)

	err := serviceManager.AddServicesToCompose("metricbeat", []string{serviceType}, env)
	if err != nil {
		log.WithFields(log.Fields{
			"service": serviceType,
			"variant": serviceVariant,
			"version": serviceVersion,
		}).Error("Could not run the service.")
	}

	mts.ServiceName = serviceType
	mts.ServiceVersion = serviceVersion

	return err
}

func (mts *MetricbeatTestSuite) thereAreEventsInTheIndex() error {
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

	result, err := retrySearch(stackName, mts.IndexName, esQuery, queryMaxAttempts, queryRetryTimeout)
	if err != nil {
		return err
	}

	return assertHitsArePresent(result, query)
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

	result, err := retrySearch(stackName, mts.IndexName, esQuery, queryMaxAttempts, queryRetryTimeout)
	if err != nil {
		return err
	}

	return assertHitsDoNotContainErrors(result, query)
}

// waitsSeconds waits for a number of seconds before the next step
func (mts *MetricbeatTestSuite) waitsSeconds(seconds string) error {
	return sleep(seconds)
}
