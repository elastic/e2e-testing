package e2e

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages-go/v10"
	"github.com/elastic/metricbeat-tests-poc/cli/config"
	"github.com/elastic/metricbeat-tests-poc/cli/services"
	log "github.com/sirupsen/logrus"
)

// StackMonitoringTestSuite represents a test suite for the stack monitoring parity tests
type StackMonitoringTestSuite struct {
	Env       map[string]string
	IndexName string
	Port      string
	Product   string
	// collection method hits
	collectionHits map[string]map[string]interface{}
}

// @collectionMethod1 the collection method to be used. Valid values: legacy, metricbeat
// @collectionMethod2 the collection method to be used. Valid values: legacy, metricbeat
func (sm *StackMonitoringTestSuite) checkDocumentsStructure(
	collectionMethod1 string, collectionMethod2 string) error {

	log.Debugf("Compare the structure of the %s documents with the structure of the %s documents", collectionMethod1, collectionMethod2)

	return assertHitsEqualStructure(
		sm.collectionHits[collectionMethod1], sm.collectionHits[collectionMethod2])
}

// checkProduct sets product name, in lowercase, setting the product port based on a validation:
// if the product is not supported, then an error is thrown
func (sm *StackMonitoringTestSuite) checkProduct(product string, collectionMethod string) error {
	sm.Product = strings.ToLower(product)

	env := map[string]string{}

	log.Debugf("Enabling %s collection, sending %s metrics to the monitoring instance", collectionMethod, sm.Product)

	if collectionMethod != "metricbeat" {
		sm.IndexName = ".monitoring-beats-7*" // stack monitoring index name for legacy collection
		/*
			method: PUT
				url: "https://{{ current_host_ip }}:{{ elasticsearch_port }}/_cluster/settings"
				body: '{ "transient": { "xpack.monitoring.collection.enabled": true } }'
				body_format: json
				validate_certs: no
				user: "{{ elasticsearch_username }}"
				password: "{{ elasticsearch_password }}"
				status_code: 200
		*/

		if product == "elasticsearch" {
			env["xpackMonitoringCollection"] = "true"
		} else {
			env["xpackMonitoring"] = "true"
			env["xpackMonitoringInterval"] = "5s"
			env["monitoringEsHost"] = "monitoringEs" // monitoring elasticsearch service name
			env["monitoringEsPort"] = "9200"         // monitoring elasticsearch port
		}
	} else {
		sm.IndexName = ".monitoring-beats-7-mb*" // stack monitoring index name for metricbeat collection
		env["xpackMonitoringCollection"] = "true"
		env["xpackMonitoring"] = "false"
	}

	switch {
	case sm.Product == "elasticsearch":
		sm.Port = strconv.Itoa(9201)
	case sm.Product == "kibana":
		sm.Port = strconv.Itoa(5602)
	case sm.Product == "logstash":
		sm.Port = strconv.Itoa(9601)
	case strings.HasSuffix(sm.Product, "beat"):
		sm.Port = strconv.Itoa(5066)

		// look up configurations under workspace's configurations directory
		dir, _ := os.Getwd()
		env[sm.Product+"ConfigFile"] = path.Join(dir, "configurations", "parity-testing", sm.Product+".yml")

		env["serviceName"] = sm.Product
	default:
		return fmt.Errorf("Product %s not supported", product)
	}

	sm.Env = env

	return nil
}

// cleanUp removes created resources
func (sm *StackMonitoringTestSuite) cleanUp() {
}

func (sm *StackMonitoringTestSuite) getCollectionMethodHits() (map[string]interface{}, error) {
	esQuery := map[string]interface{}{
		"collapse": map[string]interface{}{
			"field": "type",
		},
		"sort": map[string]interface{}{
			"timestamp": "asc",
		},
	}

	return retrySearch(sm.IndexName, esQuery, queryMaxAttempts, queryRetryTimeout)
}

func (sm *StackMonitoringTestSuite) removeProduct() {
	env := map[string]string{
		"stackVersion": stackVersion,
	}

	log.Debugf("Removing %s", sm.Product)
	srvManager := services.NewServiceManager()

	err := srvManager.RemoveServicesFromCompose("stack-monitoring", []string{sm.Product}, env)
	if err != nil {
		log.WithFields(log.Fields{
			"service": sm.Product,
		}).Error("Could not stop the service.")
	}

	sm.cleanUp()
}

func (sm *StackMonitoringTestSuite) runProduct(product string) error {
	env := map[string]string{
		"logLevel":       log.GetLevel().String(),
		product + "Tag":  stackVersion, // all products follow stack version
		product + "Port": sm.Port,      // we could run the service in another port
		"stackVersion":   stackVersion,
	}
	env = config.PutServiceEnvironment(env, product, stackVersion)

	for k, v := range sm.Env {
		env[k] = v
	}

	log.Debugf("Installing %s", sm.Product)
	srvManager := services.NewServiceManager()

	err := srvManager.AddServicesToCompose("stack-monitoring", []string{product}, env)
	if err != nil {
		log.WithFields(log.Fields{
			"port":    sm.Port,
			"service": product,
			"version": stackVersion,
		}).Error("Could not run the service.")
		return err
	}

	return nil
}

// @product the product to be installed. Valid values: elasticsearch, kibana, beats, logstash
// @collectionMethod the collection method to be used. Valid values: legacy, metricbeat
func (sm *StackMonitoringTestSuite) sendsMetricsToElasticsearch(
	product string, collectionMethod string) error {

	// validate product
	err := sm.checkProduct(product, collectionMethod)
	if err != nil {
		return err
	}

	err = sm.runProduct(product)
	if err != nil {
		return err
	}

	log.Debugf("Running %[1]s for X seconds (default: 30) to collect monitoring data internally and index it into the Monitoring index for %[1]s", product)
	sleep("30")

	log.Debugf("Stopping %s", product)
	srvManager := services.NewServiceManager()

	composes := []string{
		"stack-monitoring", // stack name
		sm.Product,         // product service
	}

	err = srvManager.RunCommand("stack-monitoring", composes, []string{"stop", sm.Product}, sm.Env)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"service": product,
			"version": stackVersion,
		}).Error("Could not stop the service.")
		return err
	}

	log.Debugf("Downloading sample documents from %s's monitoring index to a test directory", product)
	hits, err := sm.getCollectionMethodHits()
	if err != nil {
		return err
	}

	sm.collectionHits[collectionMethod] = hits

	log.Debugf("Hits: %v", hits)

	return nil
}

// StackMonitoringFeatureContext adds steps to the Godog test suite
func StackMonitoringFeatureContext(s *godog.Suite) {
	testSuite := StackMonitoringTestSuite{
		Env:            map[string]string{},
		collectionHits: map[string]map[string]interface{}{},
	}

	s.Step(`^"([^"]*)" sends metrics to Elasticsearch using the "([^"]*)" collection monitoring method$`, testSuite.sendsMetricsToElasticsearch)
	s.Step(`^the structure of the documents for the "([^"]*)" and "([^"]*)" collection are identical$`, testSuite.checkDocumentsStructure)

	s.BeforeSuite(func() {
		log.Debug("Before StackMonitoring Suite...")

		env := map[string]string{
			"stackVersion":              stackVersion,
			"xpackMonitoringCollection": "false",
		}

		log.Debug("Installing elasticsearch monitoring instance")
		startRuntimeDependencies("stack-monitoring", env, 3)
	})
	s.BeforeScenario(func(*messages.Pickle) {
		log.Debug("Before StackMonitoring Scenario...")
	})
	s.AfterSuite(func() {
		log.Debug("After StackMonitoring Suite...")
		log.Debug("Destroying elasticsearch monitoring instance, including attached services")
		tearDownRuntimeDependencies("stack-monitoring")
	})
	s.AfterScenario(func(*messages.Pickle, error) {
		log.Debug("After StackMonitoring Scenario...")
		testSuite.removeProduct()
	})
}
