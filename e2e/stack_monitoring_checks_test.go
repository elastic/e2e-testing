// +build unit

package e2e

import (
	"io/ioutil"
	"os"
	"path"
	"sort"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/stretchr/testify/assert"
)

func TestCheckKibanaParity(t *testing.T) {
	productTests := []struct {
		product              string
		monitoringCollection string
	}{
		{
			product: "elasticsearch",
		},
	}

	for _, pt := range productTests {
		sm := &StackMonitoringTestSuite{
			Env:            map[string]string{},
			collectionHits: map[string]map[string]interface{}{},
		}

		sm.checkProduct(pt.product, "legacy")
		legacy := readCollectionSample(pt.product, "legacy")

		sm.checkProduct(pt.product, "metricbeat")
		metricbeat := readCollectionSample(pt.product, "metricbeat")

		t.Run("Types length is equal for legacy and metricbeat collections", func(t *testing.T) {
			hitsPath := "hits.hits"
			legacyHits := legacy.Path(hitsPath)
			metricbeatHits := metricbeat.Path(hitsPath)

			legacyTypes, _ := checkSourceTypes(legacyHits)
			metricbeatTypes, _ := checkSourceTypes(metricbeatHits)

			assert.Equal(t, len(legacyTypes), len(metricbeatTypes))
		})

		t.Run("There are no parity errors in the test resources", func(t *testing.T) {
			err := checkParity(sm, legacy, metricbeat)
			assert.Nil(t, err)
		})
	}
}

func TestCheckSourceTypes(t *testing.T) {
	productTests := []struct {
		product               string
		monitoringCollections []string
		expectedTypes         []string
	}{
		{
			product: "elasticsearch",
			expectedTypes: []string{
				"ccr_auto_follow_stats", "cluster_stats", "enrich_coordinator_stats", "index_recovery",
				"index_stats", "indices_stats", "node_stats", "shards",
			},
		},
	}

	monitoringCollections := []string{"legacy", "metricbeat"}

	for _, pt := range productTests {
		for _, collection := range monitoringCollections {
			jsonObject := readCollectionSample(pt.product, collection)

			t.Run(pt.product, func(t *testing.T) {
				sm := StackMonitoringTestSuite{
					Env:            map[string]string{},
					collectionHits: map[string]map[string]interface{}{},
				}
				sm.checkProduct(pt.product, collection)

				hitsPath := "hits.hits"
				hits := jsonObject.Path(hitsPath)

				types, _ := checkSourceTypes(hits)
				assert.Equal(t, pt.expectedTypes, types)
			})
		}
	}
}

func TestHandleElasticsearchClusterStats_NormalisesClusterMasterNode(t *testing.T) {
	legacyDoc, metricbeatDoc := prepareElasticsearchDocs("cluster_stats")

	masterNodePath := "cluster_state.master_node"
	nodesPath := "cluster_state.nodes"
	expectedNodeName := "__normalized__"

	originalLegacyClusterMasterNode := legacyDoc.Path(masterNodePath).Data().(string)
	expectedLegacyNode := legacyDoc.Path(nodesPath + "." + originalLegacyClusterMasterNode)

	originalMetricbeatClusterMasterNode := metricbeatDoc.Path(masterNodePath).Data().(string)
	expectedMetricbeatNode := metricbeatDoc.Path(nodesPath + "." + originalMetricbeatClusterMasterNode)

	err := handleElasticsearchClusterStats(legacyDoc, metricbeatDoc)
	assert.Nil(t, err)

	// check normalized node replaced the existing one in legacy collection
	actualNode := legacyDoc.Path(nodesPath + "." + expectedNodeName)
	assert.Equal(t, expectedLegacyNode.Path("ephemeral_id").Data().(string), actualNode.Path("ephemeral_id").Data().(string))
	assert.Equal(t, expectedLegacyNode.Path("name").Data().(string), actualNode.Path("name").Data().(string))
	assert.Equal(t, expectedLegacyNode.Path("transport_address").Data().(string), actualNode.Path("transport_address").Data().(string))

	// check normalized node replaced the existing one in metricbeat collection
	actualNode = metricbeatDoc.Path(nodesPath + "." + expectedNodeName)
	assert.Equal(t, expectedMetricbeatNode.Path("ephemeral_id").Data().(string), actualNode.Path("ephemeral_id").Data().(string))
	assert.Equal(t, expectedMetricbeatNode.Path("name").Data().(string), actualNode.Path("name").Data().(string))
	assert.Equal(t, expectedMetricbeatNode.Path("transport_address").Data().(string), actualNode.Path("transport_address").Data().(string))

	assert.False(t, legacyDoc.ExistsP(nodesPath+"."+originalLegacyClusterMasterNode))
	assert.False(t, metricbeatDoc.ExistsP(nodesPath+"."+originalLegacyClusterMasterNode))

	assert.Equal(t, expectedNodeName, legacyDoc.Path(masterNodePath).Data().(string))
	assert.Equal(t, expectedNodeName, metricbeatDoc.Path(masterNodePath).Data().(string))
}

func TestHandleElasticsearchClusterStats_NormalisesXpackPolicies(t *testing.T) {
	legacyDoc, metricbeatDoc := prepareElasticsearchDocs("cluster_stats")

	err := handleElasticsearchClusterStats(legacyDoc, metricbeatDoc)
	assert.Nil(t, err)

	policyCountPath := "stack_stats.xpack.ilm.policy_count"
	assert.Equal(t, 4, metricbeatDoc.Path(policyCountPath).Data().(int))

	policyStatsPath := "stack_stats.xpack.ilm.policy_stats"
	actualPolicyStats := metricbeatDoc.Path(policyStatsPath)

	assert.Equal(t, 4, len(actualPolicyStats.Children()))

	for i := 0; i < len(actualPolicyStats.Children()); i++ {
		policyStat := actualPolicyStats.Index(i)
		phases := policyStat.Path("phases")
		if len(phases.Children()) == 1 && phases.ExistsP("hot") {
			assert.NotEqual(t, 1.0, policyStat.Path("indices_managed").Data().(float64))
		}
	}
}

func TestHandleElasticsearchClusterStats_RemovesLicenseMaxResourceUnits(t *testing.T) {
	legacyDoc, metricbeatDoc := prepareElasticsearchDocs("cluster_stats")

	err := handleElasticsearchClusterStats(legacyDoc, metricbeatDoc)
	assert.Nil(t, err)

	assert.False(t, legacyDoc.ExistsP("license.max_resource_units"))
}

func TestHandleElasticsearchClusterStats_EnsureMetricebeatFieldTypesAreInLegacy(t *testing.T) {
	legacyDoc, metricbeatDoc := prepareElasticsearchDocs("cluster_stats")

	err := handleElasticsearchClusterStats(legacyDoc, metricbeatDoc)
	assert.Nil(t, err)

	fieldTypesPath := "cluster_stats.indices.mappings.field_types"
	metricbeatFieldTypes := metricbeatDoc.Path(fieldTypesPath)
	legacyFieldTypes := legacyDoc.Path(fieldTypesPath)

	legacyChildren := legacyFieldTypes.Children()
	metricbeatChildren := metricbeatFieldTypes.Children()

	// sort by name to ensure array order
	sort.SliceStable(legacyChildren, func(i, j int) bool {
		return legacyChildren[i].Path("name").Data().(string) < legacyChildren[j].Path("name").Data().(string)
	})
	sort.SliceStable(metricbeatChildren, func(i, j int) bool {
		return metricbeatChildren[i].Path("name").Data().(string) < metricbeatChildren[j].Path("name").Data().(string)
	})

	for i, metricbeatFieldType := range metricbeatChildren {
		assert.Equal(t, metricbeatFieldType.Path("name").Data().(string), legacyChildren[i].Path("name").Data().(string))
	}
}

func TestHandleElasticsearchIndexRecovery_KeepsFirstShardOnly(t *testing.T) {
	legacy, metricbeat := prepareElasticsearchDocs("index_recovery")

	err := handleElasticsearchIndexRecovery(legacy, metricbeat)
	assert.Nil(t, err)

	legacyShards := legacy.Path("index_recovery.shards")
	metricbeatShards := metricbeat.Path("index_recovery.shards")

	assert.Equal(t, 1, len(legacyShards.Children()))
	assert.Equal(t, 0.0, legacyShards.Index(0).Path("id").Data().(float64))

	assert.Equal(t, 1, len(metricbeatShards.Children()))
	assert.Equal(t, 0.0, metricbeatShards.Index(0).Path("id").Data().(float64))
}

func TestHandleElasticsearchNodeStats_KeepsSourceNodeFields(t *testing.T) {
	expectedUUID := "expectedUUID"
	expectedName := "expectedName"
	expectedTransportAddress := "expectedTransportAddress"

	legacy, _ := gabs.ParseJSON([]byte(`{
		"source_node": {
			"uuid": "` + expectedUUID + `",
			"name": "` + expectedName + `",
			"transport_address": "` + expectedTransportAddress + `",
			"remove_me": "foo"
		}
	}`))

	err := handleElasticsearchNodeStats(legacy)
	assert.Nil(t, err)

	assert.Equal(t, expectedUUID, legacy.Path("source_node.uuid").Data().(string))
	assert.Equal(t, expectedName, legacy.Path("source_node.name").Data().(string))
	assert.Equal(t, expectedTransportAddress, legacy.Path("source_node.transport_address").Data().(string))
	assert.False(t, legacy.ExistsP("source_node.remove_me"))
}

func TestHandleElasticsearchShards_KeepsSourceNodeFields(t *testing.T) {
	expectedUUID := "expectedUUID"
	expectedName := "expectedName"

	legacy, _ := gabs.ParseJSON([]byte(`{
		"source_node": {
			"uuid": "` + expectedUUID + `",
			"name": "` + expectedName + `",
			"remove_me": "foo"
		},
		"shard": {
			"relocating_node": "relocating_node"
		}
	}`))

	err := handleElasticsearchShards(legacy)
	assert.Nil(t, err)

	assert.False(t, legacy.ExistsP("shard.relocating_node"))

	assert.Equal(t, expectedUUID, legacy.Path("source_node.uuid").Data().(string))
	assert.Equal(t, expectedName, legacy.Path("source_node.name").Data().(string))
	assert.False(t, legacy.ExistsP("source_node.remove_me"))
}

func prepareElasticsearchDocs(docType string) (*gabs.Container, *gabs.Container) {
	legacy := readCollectionSample("elasticsearch", "legacy")
	metricbeat := readCollectionSample("elasticsearch", "metricbeat")

	hitsPath := "hits.hits"
	legacyHits := legacy.Path(hitsPath)
	metricbeatHits := metricbeat.Path(hitsPath)

	_, legacySources := checkSourceTypes(legacyHits)
	_, metricbeatSources := checkSourceTypes(metricbeatHits)

	sourceValue := legacySources[docType]
	legacyDoc := gabs.Wrap(sourceValue)
	metricbeatDoc := gabs.Wrap(metricbeatSources[docType])

	return legacyDoc, metricbeatDoc
}

func readCollectionSample(product string, collectionMethod string) *gabs.Container {
	workingDir, _ := os.Getwd()

	bytes, _ := ioutil.ReadFile(path.Join(workingDir, "testresources", product+"-"+collectionMethod+"-monitoring.json"))
	jsonObj, _ := gabs.ParseJSON(bytes)

	return jsonObj
}
