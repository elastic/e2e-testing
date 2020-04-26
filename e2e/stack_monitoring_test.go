package e2e

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages-go/v10"
	"github.com/elastic/metricbeat-tests-poc/cli/config"
	log "github.com/sirupsen/logrus"
)

// StackMonitoringTestSuite represents a test suite for the stack monitoring parity tests
type StackMonitoringTestSuite struct {
	configurationFile string // the  name of the configuration file to be used in this test suite
	Env               map[string]string
	Port              string
	Product           string
}

// checkProduct sets product name, in lowercase, setting the product port based on a validation:
// if the product is not supported, then an error is thrown
func (sm *StackMonitoringTestSuite) checkProduct(product string) error {
	sm.Product = strings.ToLower(product)

	env := map[string]string{}

	switch {
	case sm.Product == "elasticsearch":
		sm.Port = strconv.Itoa(9201)
	case sm.Product == "kibana":
		sm.Port = strconv.Itoa(5602)
	case sm.Product == "logstash":
		sm.Port = strconv.Itoa(9601)
	case strings.HasSuffix(sm.Product, "beat"):
		sm.Port = strconv.Itoa(5066)

		// get latest configuration file
		configurationFileURL := "https://raw.githubusercontent.com/elastic/beats/v" + stackVersion + "/" + sm.Product + "/" + sm.Product + ".yml"

		configurationFilePath, err := downloadFile(configurationFileURL)
		if err != nil {
			return err
		}
		sm.configurationFile = configurationFilePath

		env[product+"ConfigFile"] = sm.configurationFile
		env["serviceName"] = sm.Product
		env["monitoringEsHost"] = "monitoringEs" // monitoring elasticsearch service name
	default:
		return fmt.Errorf("Product %s not supported", product)
	}

	sm.Env = env

	return nil
}

// cleanUp removes created resources
func (sm *StackMonitoringTestSuite) cleanUp() {
	if _, err := os.Stat(sm.configurationFile); err == nil {
		err = os.Remove(sm.configurationFile)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"product": sm.Product,
				"path":    sm.configurationFile,
			}).Warn("Configuration file was not removed.")

			return
		}

		log.WithFields(log.Fields{
			"product": sm.Product,
			"path":    sm.configurationFile,
		}).Debug("Configuration file removed.")
	}
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

	sm.cleanUp()
}

func (sm *StackMonitoringTestSuite) runProduct(product string) error {
	env := map[string]string{
		product + "Tag":  stackVersion, // all products follow stack version
		product + "Port": sm.Port,      // we could run the service in another port
	}
	env = config.PutServiceEnvironment(env, product, stackVersion)

	for k, v := range sm.Env {
		env[k] = v
	}

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
	testSuite := StackMonitoringTestSuite{
		Env: map[string]string{},
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
