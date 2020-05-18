package e2e

import (
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages-go/v10"
	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/cli/services"
	log "github.com/sirupsen/logrus"
)

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

// StackMonitoringTestSuite represents a test suite for the stack monitoring parity tests
type StackMonitoringTestSuite struct {
	Env       map[string]string
	IndexName string
	Port      int
	Product   string
	// collection method hits
	collectionHits map[string]map[string]interface{}
	// extra fields used during assertions
	allowedDeletionsExtra  []string
	allowedInsertionsExtra []string
	handleSpecialCases     func(docType string, legacy *gabs.Container, metricbeat *gabs.Container) error
}

// @collectionMethod1 the collection method to be used. Valid values: legacy, metricbeat
// @collectionMethod2 the collection method to be used. Valid values: legacy, metricbeat
func (sm *StackMonitoringTestSuite) checkDocumentsStructure(
	collectionMethod1 string, collectionMethod2 string) error {

	log.Debugf("Compare the structure of the %s documents with the structure of the %s documents", collectionMethod1, collectionMethod2)

	return assertHitsEqualStructure(sm, collectionMethod1, collectionMethod2)
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

	productIndexID := sm.Product

	// look up configurations under workspace's configurations directory
	workingDir, _ := os.Getwd()

	switch {
	case sm.Product == "elasticsearch":
		sm.Port = 9201

		productIndexID = "es"

		sm.handleSpecialCases = func(docType string, legacy *gabs.Container, metricbeat *gabs.Container) error {
			if docType == "index_recovery" {
				return handleElasticsearchIndexRecovery(legacy, metricbeat)
			}

			if docType == "cluster_stats" {
				return handleElasticsearchClusterStats(legacy, metricbeat)
			}

			if docType == "node_stats" {
				return handleElasticsearchNodeStats(legacy, metricbeat)
			}

			if docType == "shards" {
				return handleElasticsearchShards(legacy, metricbeat)
			}

			return nil
		}
	case sm.Product == "kibana":
		sm.allowedDeletionsExtra = []string{
			"kibana_stats.response_times.max",
			"kibana_stats.response_times.average",
		}

		sm.handleSpecialCases = func(docType string, legacy *gabs.Container, metricbeat *gabs.Container) error {
			if docType == "kibana_settings" {
				return handleKibanaLegacySettings(legacy)
			}

			return nil
		}
	case sm.Product == "logstash":
		sm.handleSpecialCases = func(docType string, legacy *gabs.Container, metricbeat *gabs.Container) error {
			if docType == "logstash_stats" {
				return handleLogstashStats(sm.Product, legacy, metricbeat)
			}

			return nil
		}

		env[sm.Product+"PipelinesPath"] = path.Join(workingDir, "configurations", "parity-testing", "pipelines")
	case strings.HasSuffix(sm.Product, "beat"):
		productIndexID = "beats"

		sm.handleSpecialCases = func(docType string, legacy *gabs.Container, metricbeat *gabs.Container) error {
			if docType == "beats_stats" {
				return handleBeatsStats(legacy, metricbeat)
			}

			return nil
		}

		env[sm.Product+"ConfigFile"] = path.Join(workingDir, "configurations", "parity-testing", sm.Product+".yml")

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

	sm.IndexName = ".monitoring-" + productIndexID + "-7"
	if collectionMethod == "metricbeat" {
		sm.IndexName += "-mb"
	}
	t := time.Now()
	sm.IndexName += "-" + t.Format("2006.01.02") // match monitoring index name format

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

	// look up configurations under workspace's configurations directory
	dir, _ := os.Getwd()
	env["metricbeatConfigFile"] = path.Join(dir, "configurations", "metricbeat", "monitoring-"+sm.Product+".yml")

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

	return nil
}

func (sm *StackMonitoringTestSuite) runProduct(product string, collectionMethod string) error {
	env := map[string]string{
		"logLevel":       log.GetLevel().String(),
		product + "Tag":  stackVersion,          // all products follow stack version
		product + "Port": strconv.Itoa(sm.Port), // we could run the service in another port
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

	if sm.Product == "elasticsearch" {
		// the production instance of elasticsearch binds its port in localhost
		_, err = waitForElasticsearchFromHostPort("localhost", sm.Port, (3 * time.Minute))
		if err != nil {
			return err
		}
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

	defer func(ctx context.Context) {
		log.Debugf("Deleting monitoring index %s", sm.IndexName)
		if err := deleteIndex(ctx, sm.IndexName); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"index": sm.IndexName,
			}).Warn("The monitoring index was not deleted, but we are not failing the test case")
		}
	}(context.Background())

	// validate product
	err := sm.checkProduct(product, collectionMethod)
	if err != nil {
		return err
	}

	err = sm.runProduct(product, collectionMethod)
	if err != nil {
		return err
	}

	log.Debugf("Running %[1]s for X seconds (default: 60) to collect monitoring data internally and index it into the Monitoring index for %[1]s", product)
	sleep("60")

	composes := []string{
		sm.Product, // product service
	}

	if collectionMethod == "metricbeat" {
		composes = append(composes, "metricbeat")

		if log.IsLevelEnabled(log.DebugLevel) {
			composes := []string{
				"stack-monitoring", // stack name
				"metricbeat",       // metricbeat service
			}

			err = serviceManager.RunCommand("stack-monitoring", composes, []string{"logs", "metricbeat"}, map[string]string{})
			if err != nil {
				log.WithFields(log.Fields{
					"error":             err,
					"metricbeatVersion": stackVersion,
					"product":           sm.Product,
					"serviceVersion":    stackVersion,
				}).Error("Could not retrieve Metricbeat logs")
			}
		}
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

	log.Debugf("Fetching sample documents from %s's monitoring index", product)
	hits, err := sm.getCollectionMethodHits()
	if err != nil {
		return err
	}

	sm.collectionHits[collectionMethod] = hits

	return nil
}

// arrayContains checks that the array contains the field, or
// the fiels starts with any of the values from the array
func arrayContainsField(arr []string, field string) bool {
	for _, a := range arr {
		if a == field || strings.HasPrefix(field, a+".") {
			return true
		}
	}

	return false
}

func arrayDiff(a []string, b []string) []string {
	target := map[string]bool{}
	for _, x := range b {
		target[x] = true
	}

	result := []string{}
	for _, x := range a {
		if _, ok := target[x]; !ok {
			result = append(result, x)
		}
	}

	return result
}

// assertHitsEqualStructure returns an error if hits don't share structure
func assertHitsEqualStructure(sm *StackMonitoringTestSuite, collectionMethod1 string, collectionMethod2 string) error {
	collectionMethod1Hits := sm.collectionHits[collectionMethod1]
	collectionMethod2Hits := sm.collectionHits[collectionMethod2]

	collectionMethod1HitsJSON := gabs.Wrap(collectionMethod1Hits)
	collectionMethod2HitsJSON := gabs.Wrap(collectionMethod2Hits)

	err := checkParity(sm, collectionMethod1HitsJSON, collectionMethod2HitsJSON)
	if err != nil {
		return err
	}

	return nil
}

func checkParity(sm *StackMonitoringTestSuite, legacyContainer *gabs.Container, metricbeatContainer *gabs.Container) error {
	allowedInsertionsInMetricbeatDocs := []string{
		"service",
		"@timestamp",
		"agent",
		"event",
		"host",
		"ecs",
		"metricset",
	}
	allowedInsertionsInMetricbeatDocs = append(allowedInsertionsInMetricbeatDocs, sm.allowedInsertionsExtra...)

	allowedDeletionsFromMetricbeatDocs := []string{
		"source_node",
	}
	allowedDeletionsFromMetricbeatDocs = append(allowedDeletionsFromMetricbeatDocs, sm.allowedDeletionsExtra...)

	hitsPath := "hits.hits"
	legacyHits := legacyContainer.Path(hitsPath)
	metricbeatHits := metricbeatContainer.Path(hitsPath)

	legacyTypes, legacySources := checkSourceTypes(legacyHits)
	metricbeatTypes, metricbeatSources := checkSourceTypes(metricbeatHits)

	if len(legacyTypes) > len(metricbeatTypes) {
		// returns an array as the result of removing all elements in 'b' from the 'a' array
		// only used here
		diff := arrayDiff(legacyTypes, metricbeatTypes)

		return fmt.Errorf("Found more legacy-indexed document types than metricbeat-indexed document types. Document types indexed by legacy collection but not by Metricbeat collection: %v", diff)
	}

	foundErrors := []error{}
	for docType, sourceValue := range legacySources {
		legacyDoc := gabs.Wrap(sourceValue)
		metricbeatDoc := gabs.Wrap(metricbeatSources[docType])

		err := sm.handleSpecialCases(docType, legacyDoc, metricbeatDoc)
		if err != nil {
			foundErrors = append(foundErrors, err)
		}

		unexpectedInsertions := []string{}
		unexpectedDeletions := []string{}

		// Flatten a JSON array or object into an object of key/value pairs for each
		// field, where the key is the full path of the structured field in dot path
		// notation matching the spec for the method Path.
		flatLegacy, err := legacyDoc.Flatten()
		if err != nil {
			foundErrors = append(foundErrors, fmt.Errorf("Error flattening legacy doc for %s: %v - %v", docType, legacyDoc, err))
			flatLegacy = map[string]interface{}{}
		}

		flatMetricbeat, err := metricbeatDoc.Flatten()
		if err != nil {
			foundErrors = append(foundErrors, fmt.Errorf("Error flattening metricbeat doc for %s: %v - %v", docType, metricbeatDoc, err))
			flatMetricbeat = map[string]interface{}{}
		}

		for k := range flatMetricbeat {
			if _, ok := flatLegacy[k]; !ok {
				if !arrayContainsField(allowedInsertionsInMetricbeatDocs, k) {
					unexpectedInsertions = append(unexpectedInsertions, k)
				}

				continue
			}
		}

		for k := range flatLegacy {
			if _, ok := flatMetricbeat[k]; !ok {
				if !arrayContainsField(allowedDeletionsFromMetricbeatDocs, k) {
					unexpectedDeletions = append(unexpectedDeletions, k)
				}

				continue
			}
		}

		if len(unexpectedInsertions) == 0 && len(unexpectedDeletions) == 0 {
			log.WithFields(log.Fields{
				"docType": docType,
				"product": sm.Product,
			}).Info("Expected parity between Metricbeat-indexed doc and legacy-indexed doc found.")
			continue
		}

		if len(unexpectedInsertions) > 0 {
			for _, insertion := range unexpectedInsertions {
				err := fmt.Errorf("Metricbeat-indexed doc for type='%s' has unexpected insertion", docType)
				log.WithFields(log.Fields{
					"docType":   docType,
					"insertion": insertion,
					"product":   sm.Product,
				}).Warn(err.Error())
				foundErrors = append(foundErrors, err)
			}
		}

		if len(unexpectedDeletions) > 0 {
			for _, deletion := range unexpectedDeletions {
				err := fmt.Errorf("Metricbeat-indexed doc for type='%s' has unexpected deletion", docType)
				log.WithFields(log.Fields{
					"docType":  docType,
					"deletion": deletion,
					"product":  sm.Product,
				}).Warn(err.Error())
				foundErrors = append(foundErrors, err)
			}
		}
	}

	if len(foundErrors) > 0 {
		return fmt.Errorf("Found %d errors while checking parity", len(foundErrors))
	}

	log.Info("No parity errors found!")

	return nil
}

// checkSourceTypes returns an array of types present in the document, plus a map with _source documents,
// indexed by document type
func checkSourceTypes(container *gabs.Container) ([]string, map[string]interface{}) {
	types := []string{}
	sources := map[string]interface{}{}

	for i := 0; i < len(container.Children()); i++ {
		containerChild := container.Index(i)

		t, _ := gabs.New().Set(containerChild.Path("_source.type").Data())
		data := t.Data().(string)

		types = append(types, data)

		source, _ := gabs.New().Set(containerChild.Path("_source").Data())
		sources[data] = source.Data()
	}

	return types, sources
}
