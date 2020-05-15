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
	Port      string
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

	switch {
	case sm.Product == "elasticsearch":
		sm.Port = strconv.Itoa(9201)

		productIndexID = "es"

		sm.handleSpecialCases = func(docType string, legacy *gabs.Container, metricbeat *gabs.Container) error {
			if docType == "index_recovery" {
				// Normalize `index_recovery.shards` array field to have only one object in it.
				shardsPath := "index_recovery.shards"

				legacyShards := legacy.Path(shardsPath)
				metricbeatShards := metricbeat.Path(shardsPath)

				legacyShards = legacyShards.Index(0)
				metricbeatShards = metricbeatShards.Index(0)

				return nil
			}

			if docType == "cluster_stats" {
				// We expect the node ID to be different in the internally-collected vs. metricbeat-collected
				// docs because the tests spin up a fresh 1-node cluster prior to each type of collection.
				// So we normalize the node names.
				masterNodePath := "cluster_state.master_node"
				nodesPath := "cluster_state.nodes"
				newNodeName := "__normalized__"

				origNodeName := legacy.Path(masterNodePath).String()
				legacy.SetP(newNodeName, masterNodePath)
				metricbeat.SetP(newNodeName, masterNodePath)

				legacy.SetP(legacy.Path(nodesPath+"."+origNodeName), nodesPath+"."+newNodeName)
				metricbeat.SetP(metricbeat.Path(nodesPath+"."+origNodeName), nodesPath+"."+newNodeName)

				legacy.DeleteP(nodesPath + "." + origNodeName)
				metricbeat.DeleteP(nodesPath + "." + origNodeName)

				// When Metricbeat-based monitoring is used, Metricbeat will setup an ILM policy for
				// metricbeat-* indices. Obviously this policy is not present when internal monitoring is
				// used, since Metricbeat is not running in that case. So we normalize by removing the
				// usage stats associated with the Metricbeat-created ILM policy.
				policyStatsPath := "stack_stats.xpack.ilm.policy_stats"
				metricbeatPolicyStats := metricbeat.Path(policyStatsPath)

				// The Metricbeat ILM policy is the one with exactly one phase: hot
				newPolicyStats := []*gabs.Container{}
				for i := 0; i < len(metricbeatPolicyStats.Children()); i++ {
					policyStat := metricbeatPolicyStats.Index(i)
					policyPhasesContainer := policyStat.Path("phases")
					policyPhases := policyPhasesContainer.Data().(map[string]interface{})
					if len(policyPhases) == 1 &&
						policyPhasesContainer.Index(0).Data() == "hot" &&
						policyStat.Path("indices_managed").Data() == 1 {

						continue
					} else {
						newPolicyStats = append(newPolicyStats, policyStat)
					}
				}

				metricbeat.SetP(newPolicyStats, "stack_stats.xpack.ilm.policy_stats")
				metricbeat.SetP(len(newPolicyStats), "stack_stats.xpack.ilm.policy_count")

				// Metricbeat modules will automatically strip out keys that contain a null value
				// and `license.max_resource_units` is only available on certain license levels.
				// The `_cluster/stats` api will return a `null` entry for this key if the license level
				// does not have a `max_resouce_units` which causes Metricbeat to strip it out
				// If that happens, just assume parity between the two
				maxResourceUnitsPath := "license.max_resource_units"
				if legacy.ExistsP(maxResourceUnitsPath) {
					legacy.DeleteP(maxResourceUnitsPath)
				}

				// The `field_types` field returns a list of what field types exist in all existing mappings
				// When running the parity tests, it is likely that the indices change between when we query
				// internally collected documents versus when we query Metricbeat collected documents. These
				// two may or may not match as a result.
				// To get around this, we know that the parity tests query internally collected documents first
				// so we will ensure that all `field_types` that exist from that source also exist in the
				// Metricbeat `field_types` (It is very likely the Metricbeat `field_types` will contain more)
				internalContainsAllInMetricbeat := false
				fieldTypesPath := "cluster_stats.indices.mappings.field_types"
				if legacy.ExistsP(fieldTypesPath) {
					legacyFieldTypes := legacy.Path(fieldTypesPath)
					metricbeatFieldTypes := metricbeat.Path(fieldTypesPath)
					for i := 0; i < len(legacyFieldTypes.Children()); i++ {
						legacyFieldType := legacyFieldTypes.Index(i)
						legacyFieldTypeName := legacyFieldType.Path("name")
						found := false
						for j := 0; j < len(metricbeatFieldTypes.Children()); j++ {
							metricbeatFieldType := metricbeatFieldTypes.Index(j)

							metricbeatFieldTypeName := metricbeatFieldType.Path("name")
							if metricbeatFieldTypeName.Data() == legacyFieldTypeName.Data() {
								found = true
							}
						}

						if !found {
							break
						}

						internalContainsAllInMetricbeat = true
					}

					if internalContainsAllInMetricbeat {
						legacy.SetP(metricbeat.Path(fieldTypesPath), fieldTypesPath)
					}
				}

				return nil
			}

			if docType == "node_stats" {
				// Metricbeat-indexed docs of `type:node_stats` fake the `source_node` field since its required
				// by the UI. However, it only fakes the `source_node.uuid`, `source_node.name`, and
				// `source_node.transport_address` fields since those are the only ones actually used by
				// the UI. So we normalize by removing all but those three fields from the internally-indexed
				// doc.
				sourceNode := legacy.Path("source_node")
				newSourceNode := gabs.New()
				newSourceNode.SetP(sourceNode.Path("uuid"), "uuid")
				newSourceNode.SetP(sourceNode.Path("name"), "name")
				newSourceNode.SetP(sourceNode.Path("transport_address"), "transport_address")
				legacy.SetP(newSourceNode, "source_node")

				return nil
			}

			if docType == "shards" {
				// Metricbeat-indexed docs of `type:shard` fake the `source_node` field since its required
				// by the UI. However, it only fakes the `source_node.uuid` and `source_node.name` fields
				// since those are the only ones actually used by the UI. So we normalize by removing all
				// but those two fields from the internally-indexed doc.
				sourceNode := legacy.Path("source_node")
				newSourceNode := gabs.New()
				newSourceNode.SetP(sourceNode.Path("uuid"), "uuid")
				newSourceNode.SetP(sourceNode.Path("name"), "name")
				legacy.SetP(newSourceNode, "source_node")

				// Internally-indexed docs of `type:shard` will set `shard.relocating_node` to `null`, if
				// the shard is not relocating. However, Metricbeat-indexed docs of `type:shard` will simply
				// not send the `shard.relocating_node` field if the shard is not relocating. So we normalize
				// by deleting the `shard.relocating_node` field from the internally-indexed doc if the shard
				// is not relocating.
				legacy.DeleteP("shards.relocating_node")

				return nil
			}

			return nil
		}
	case sm.Product == "kibana":
		sm.allowedDeletionsExtra = []string{
			"kibana_stats.response_times.max",
			"kibana_stats.response_times.average",
		}

		sm.handleSpecialCases = func(docType string, legacy *gabs.Container, metricbeat *gabs.Container) error {
			// Internal collection will index kibana_settings.xpack.default_admin_email as null
			// whereas Metricbeat collection simply won't index it. So if we find kibana_settings.xpack.default_admin_email
			// is null, we simply remove it
			if docType == "kibana_settings" {
				err := legacy.DeleteP("kibana_settings.xpack.default_admin_email")
				if err != nil {
					return fmt.Errorf("Could not remove default_admin_email field: %v", err)
				}
			}

			return nil
		}
	case sm.Product == "logstash":
		sm.handleSpecialCases = func(docType string, legacy *gabs.Container, metricbeat *gabs.Container) error {
			pipelinesPath := "logstash_stats.pipelines"

			if docType == "logstash_stats" {
				legacyPipelines := legacy.Path(pipelinesPath)
				metricbeatPipelines := metricbeat.Path(pipelinesPath)

				legacyPipeline := legacyPipelines.Index(0)
				metricbeatPipeline := metricbeatPipelines.Index(0)

				legacyVertices := legacyPipeline.Path("vertices")
				metricbeatVertices := metricbeatPipeline.Path("vertices")

				// no need to sort, as the comparison will be made key by key

				foundError := false
				if legacyVertices == nil {
					foundError = true
					log.WithFields(log.Fields{
						"product": sm.Product,
					}).Warn(pipelinesPath + ".0.vertices is null for legacy collection")
				}
				if metricbeatVertices == nil {
					foundError = true
					log.WithFields(log.Fields{
						"product": sm.Product,
					}).Warn(pipelinesPath + ".0.vertices is null for metricbeat collection")
				}
				if foundError {
					return fmt.Errorf("%s.0.vertices for legacy or metricbeat collection is null", pipelinesPath)
				}
			}

			return nil
		}
	case strings.HasSuffix(sm.Product, "beat"):
		productIndexID = "beats"

		// When Metricbeat monitors Filebeat, it encounters a different set of file IDs in
		// `type:beats_stats` documents than when internal collection monitors Filebeat. However,
		// we expect the _number_ of files being harvested by Filebeat in either case to match.
		// If the numbers match we normalize the file lists in `type:beats_stats` docs collected
		// by both methods so their parity comparison succeeds.
		sm.handleSpecialCases = func(docType string, legacy *gabs.Container, metricbeat *gabs.Container) error {
			filesPath := "beats_stats.metrics.filebeat.harvester.files"

			if docType == "beats_stats" {
				legacyFiles := legacy.Path(filesPath)
				metricbeatFiles := metricbeat.Path(filesPath)

				legacyFilesCount := len(legacyFiles.Children())
				metricbeatFilesCount := len(metricbeatFiles.Children())

				if legacyFilesCount != metricbeatFilesCount {
					return fmt.Errorf("The number of harvested files in legacy (%d) and metricbeat (%d) collection is different", legacyFilesCount, metricbeatFilesCount)
				}

				log.Debugf("The number of harvested files in legacy and metricbeat collection is the same: %d", legacyFilesCount)

				legacy.DeleteP(filesPath)
				metricbeat.DeleteP(filesPath)
			}

			return nil
		}

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

		return fmt.Errorf("Found more legacy-indexed document types than metricbeat-indexed document types. Document types indexed by internal collection but not by Metricbeat collection: %v", diff)
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

	for _, containerChild := range container.Children() {
		t, _ := gabs.New().Set(containerChild.Path("_source.type").Data())
		data := t.Data().(string)

		types = append(types, data)

		source, _ := gabs.New().Set(containerChild.Path("_source").Data())
		sources[data] = source.Data()
	}

	return types, sources
}
