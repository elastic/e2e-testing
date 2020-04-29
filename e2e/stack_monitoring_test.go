package e2e

import (
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

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

	env := map[string]string{
		"monitoringEsHost": "monitoringEs", // monitoring elasticsearch service name
		"monitoringEsPort": "9200",         // monitoring elasticsearch port
	}

	log.Debugf("Enabling %s collection, sending %s metrics to the monitoring instance", collectionMethod, sm.Product)

	sm.IndexName = ".monitoring-beats-7"
	if collectionMethod == "metricbeat" {
		sm.IndexName += "-mb"
	}
	t := time.Now()
	sm.IndexName += "-" + t.Format("2006.01.02") // match monitoring index name format

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

		if collectionMethod == "metricbeat" {
			env["httpEnabled"] = "true"
			env["httpPort"] = "5066"
			env["xpackMonitoring"] = "false"
		} else {
			env["xpackMonitoring"] = "true"
		}
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

	err := srvManager.RemoveServicesFromCompose("stack-monitoring", []string{sm.Product, "metricbeat"}, env)
	if err != nil {
		log.WithFields(log.Fields{
			"service": sm.Product,
		}).Error("Could not stop the service.")
	}

	sm.cleanUp()
}

// runMetricbeat runs a metricbeat service for monitoring a product
func (sm *StackMonitoringTestSuite) runMetricbeat() error {
	serviceManager := services.NewServiceManager()

	env := map[string]string{
		"BEAT_STRICT_PERMS": "false",
		"logLevel":          log.GetLevel().String(),
		"metricbeatTag":     stackVersion,
		"stackVersion":      stackVersion,
		sm.Product + "Tag":  stackVersion,
		"serviceName":       sm.Product,
	}

	if strings.HasSuffix(sm.Product, "beat") {
		// look up configurations under workspace's configurations directory
		dir, _ := os.Getwd()
		env["metricbeatConfigFile"] = path.Join(dir, "configurations", "metricbeat", "beat-xpack.yml")
	}

	for k, v := range env {
		sm.Env[k] = v
	}

	err := serviceManager.AddServicesToCompose("stack-monitoring", []string{"metricbeat"}, sm.Env)
	if err != nil {
		log.WithFields(log.Fields{
			"error":             err,
			"metricbeatVersion": stackVersion,
			"product":           sm.Product,
			"productVersion":    stackVersion,
		}).Error("Could not run Metricbeat for the service")

		return err
	}

	log.WithFields(log.Fields{
		"metricbeatVersion": stackVersion,
		"product":           sm.Product,
		"serviceVersion":    stackVersion,
	}).Info("Metricbeat is running configured for the product")

	if log.IsLevelEnabled(log.DebugLevel) {
		composes := []string{
			"stack-monitoring", // stack name
			"metricbeat",       // metricbeat service
		}

		err = serviceManager.RunCommand("stack-monitoring", composes, []string{"logs", "metricbeat"}, env)
		if err != nil {
			log.WithFields(log.Fields{
				"error":             err,
				"metricbeatVersion": stackVersion,
				"product":           sm.Product,
				"serviceVersion":    stackVersion,
			}).Error("Could not retrieve Metricbeat logs")

			return err
		}
	}

	return nil
}

func (sm *StackMonitoringTestSuite) runProduct(product string, collectionMethod string) error {
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

	if collectionMethod == "metricbeat" {
		err = sm.runMetricbeat()
		if err != nil {
			return err
		}
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

	err = sm.runProduct(product, collectionMethod)
	if err != nil {
		return err
	}

	log.Debugf("Running %[1]s for X seconds (default: 30) to collect monitoring data internally and index it into the Monitoring index for %[1]s", product)
	sleep("30")

	composes := []string{
		sm.Product, // product service
	}

	if collectionMethod == "metricbeat" {
		composes = append(composes, "metricbeat")
	}

	log.Debugf("Stopping %s", product)
	srvManager := services.NewServiceManager()

	err = srvManager.RemoveServicesFromCompose("stack-monitoring", composes, sm.Env)
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

	log.Debugf("Deleting monitoring index %s", sm.IndexName)
	fn := func(ctx context.Context) {
		err := deleteIndex(ctx, sm.IndexName)
		if err != nil {
			log.WithFields(log.Fields{
				"index": sm.IndexName,
			}).Warn("The monitoring index was not deleted, but we are not failing the test case")
		}
	}
	defer fn(context.Background())

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
