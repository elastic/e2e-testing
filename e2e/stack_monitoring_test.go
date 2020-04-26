package e2e

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages-go/v10"
	"github.com/elastic/metricbeat-tests-poc/cli/config"
	log "github.com/sirupsen/logrus"
)

// StackMonitoringTestSuite represents a test suite for the stack monitoring parity tests
type StackMonitoringTestSuite struct {
	Port    string
	Product string
}

// checkProduct sets product name, in lowercase, setting the product port based on a validation:
// if the product is not supported, then an error is thrown
func (sm *StackMonitoringTestSuite) checkProduct(product string) error {
	sm.Product = strings.ToLower(product)

	switch sm.Product {
	case "elasticsearch":
		sm.Port = strconv.Itoa(9201)
	case "kibana":
		sm.Port = strconv.Itoa(5602)
	case "logstash":
		sm.Port = strconv.Itoa(9601)
	default:
		return fmt.Errorf("Product %s not supported", product)
	}

	return nil
}

func (sm *StackMonitoringTestSuite) removeProduct() {
	env := map[string]string{
		"stackVersion": stackVersion,
	}

	log.Debugf("Removing %s", sm.Product)
	err := serviceManager.RemoveServicesFromCompose("stack-monitoring", []string{sm.Product}, env)
	if err != nil {
		log.WithFields(log.Fields{
			"service": sm.Product,
		}).Error("Could not stop the service.")
	}
}

func (sm *StackMonitoringTestSuite) runProduct(product string) error {
	env := map[string]string{
		product + "Tag":  stackVersion, // all products follow stack version
		product + "Port": sm.Port,      // we could run the service in another port
	}
	env = config.PutServiceEnvironment(env, product, stackVersion)

	log.Debugf("Installing %s", sm.Product)
	err := serviceManager.AddServicesToCompose("stack-monitoring", []string{product}, env)
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
	err := sm.checkProduct(product)
	if err != nil {
		return err
	}

	err = sm.runProduct(product)
	if err != nil {
		return err
	}

	envVars := map[string]string{}
	if collectionMethod == "metricbeat" {
		log.Debugf("Installing metricbeat configured for %s to send metrics to the elasticsearch monitoring instance", product)
	} else {
		log.Debugf("Enabling %s collection, sending metrics to the monitoring instance", collectionMethod)

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
			envVars["xpack.monitoring.collection.enabled"] = "true"
		}
	}
	log.Debugf("Running %[1]s for X seconds (default: 30) to collect monitoring data internally and index it into the Monitoring index for %[1]s", product)
	log.Debugf("Stopping %s", product)
	log.Debugf("Downloading sample documents from %s's monitoring index to a test directory", product)
	log.Debugf("Disable %s", collectionMethod)

	return godog.ErrPending
}

// @collectionMethod1 the collection method to be used. Valid values: legacy, metricbeat
// @collectionMethod2 the collection method to be used. Valid values: legacy, metricbeat
func (sm *StackMonitoringTestSuite) checkDocumentsStructure(
	collectionMethod1 string, collectionMethod2 string) error {

	log.Debugf("Compare the structure of the %s documents with the structure of the %s documents", collectionMethod1, collectionMethod2)

	return godog.ErrPending
}

// StackMonitoringFeatureContext adds steps to the Godog test suite
func StackMonitoringFeatureContext(s *godog.Suite) {
	testSuite := StackMonitoringTestSuite{}

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
