package e2e

import (
	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages-go/v10"
	log "github.com/sirupsen/logrus"
)

// StackMonitoringTestSuite represents a test suite for the stack monitoring parity tests
type StackMonitoringTestSuite struct {
}

// @product the product to be installed. Valid values: elasticsearch, kibana, beats, logstash
// @collectionMethod the collection method to be used. Valid values: legacy, metricbeat
func (sm *StackMonitoringTestSuite) sendsMetricsToElasticsearch(
	product string, collectionMethod string) error {

	log.Debugf("Installing %s", product)
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
		log.Debug("Installing elasticsearch monitoring instance")
	})
	s.BeforeScenario(func(*messages.Pickle) {
		log.Debug("Before StackMonitoring Scenario...")
	})
	s.AfterSuite(func() {
		log.Debug("After StackMonitoring Suite...")
		log.Debug("Destroying elasticsearch monitoring instance, including attached services")
	})
	s.AfterScenario(func(*messages.Pickle, error) {
		log.Debug("After StackMonitoring Scenario...")
	})
}
