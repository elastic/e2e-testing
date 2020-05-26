package e2e

import (
	"regexp"
	"sort"

	"github.com/Jeffail/gabs/v2"
)

// checkMapKeysWithoutArrayIndices remove all keys producued by array indices
// i.e. 'stack_stats.xpack.ilm.policy_stats.3.phases.hot.actions.0'
func checkMapKeysWithoutArrayIndices(keysMap map[string]interface{}) map[string]bool {
	keysWithoutArrayIndices := map[string]bool{}
	for k := range keysMap {
		validKey := regexp.MustCompile(`\b.\d+?\b`)
		keyWithoutArrayIndices := validKey.ReplaceAllLiteralString(k, "")
		keysWithoutArrayIndices[keyWithoutArrayIndices] = true
	}

	return keysWithoutArrayIndices
}

// checkSourceTypes returns an array of types present in the document, alphabetically sorted,
// plus a map with _source documents, indexed by document type
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

	sort.SliceStable(types, func(i, j int) bool {
		return types[i] < types[j]
	})

	return types, sources
}

// handleElasticsearchClusterStats
func handleElasticsearchClusterStats(legacy *gabs.Container, metricbeat *gabs.Container) error {
	// We expect the node ID to be different in the internally-collected vs. metricbeat-collected
	// docs because the tests spin up a fresh 1-node cluster prior to each type of collection.
	// So we normalize the node names.
	masterNodePath := "cluster_state.master_node"
	nodesPath := "cluster_state.nodes"
	newNodeName := "__normalized__"

	origNodeName := legacy.Path(masterNodePath).Data().(string)
	legacy.SetP(newNodeName, masterNodePath)
	legacy.SetP(legacy.Path(nodesPath+"."+origNodeName).Data(), nodesPath+"."+newNodeName)
	legacy.DeleteP(nodesPath + "." + origNodeName)

	origNodeName = metricbeat.Path(masterNodePath).Data().(string)
	metricbeat.SetP(newNodeName, masterNodePath)
	metricbeat.SetP(metricbeat.Path(nodesPath+"."+origNodeName).Data(), nodesPath+"."+newNodeName)
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
		policyPhases := policyPhasesContainer.ChildrenMap()
		if len(policyPhases) == 1 &&
			policyPhasesContainer.ExistsP("hot") &&
			policyStat.Path("indices_managed").Data().(float64) == 1.0 {

			continue
		} else {
			newPolicyStats = append(newPolicyStats, policyStat)
		}
	}

	metricbeat.DeleteP(policyStatsPath)
	metricbeat.ArrayP(policyStatsPath)
	for _, p := range newPolicyStats {
		metricbeat.ArrayAppendP(p.Data(), policyStatsPath)
	}

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
	legacyContainsAllInMetricbeat := false
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

			legacyContainsAllInMetricbeat = true
		}

		if legacyContainsAllInMetricbeat {
			legacy.DeleteP(fieldTypesPath)
			legacy.ArrayP(fieldTypesPath)
			for i := 0; i < len(metricbeatFieldTypes.Children()); i++ {
				metricbeatFieldType := metricbeatFieldTypes.Index(i)
				legacy.ArrayAppendP(metricbeatFieldType.Data(), fieldTypesPath)
			}
		}
	}

	return nil
}

// handleElasticsearchIndexRecovery Normalize `index_recovery.shards` array field to have only one object in it.
func handleElasticsearchIndexRecovery(legacy *gabs.Container, metricbeat *gabs.Container) error {
	shardsPath := "index_recovery.shards"

	legacyShards := legacy.Path(shardsPath)
	metricbeatShards := metricbeat.Path(shardsPath)

	firstLegacyShard := legacyShards.Index(0)
	legacy.DeleteP(shardsPath)
	legacy.ArrayP(shardsPath)
	legacy.ArrayAppendP(firstLegacyShard.Data(), shardsPath)

	firstMetricbeatShard := metricbeatShards.Index(0)
	metricbeat.DeleteP(shardsPath)
	metricbeat.ArrayP(shardsPath)
	metricbeat.ArrayAppendP(firstMetricbeatShard.Data(), shardsPath)

	return nil
}

// handleElasticsearchNodeStats
// Metricbeat-indexed docs of `type:node_stats` fake the `source_node` field since its required
// by the UI. However, it only fakes the `source_node.uuid`, `source_node.name`, and
// `source_node.transport_address` fields since those are the only ones actually used by
// the UI. So we normalize by removing all but those three fields from the internally-indexed doc.
func handleElasticsearchNodeStats(legacy *gabs.Container) error {
	sourceNode := legacy.Path("source_node")

	uuid := sourceNode.Path("uuid").Data().(string)
	name := sourceNode.Path("name").Data().(string)
	transportAddress := sourceNode.Path("transport_address").Data().(string)

	legacy.DeleteP("source_node")
	legacy.SetP(uuid, "source_node.uuid")
	legacy.SetP(name, "source_node.name")
	legacy.SetP(transportAddress, "source_node.transport_address")

	return nil
}

// handleElasticsearchShards
// Metricbeat-indexed docs of `type:shard` fake the `source_node` field since its required
// by the UI. However, it only fakes the `source_node.uuid` and `source_node.name` fields
// since those are the only ones actually used by the UI. So we normalize by removing all
// but those two fields from the internally-indexed doc.
func handleElasticsearchShards(legacy *gabs.Container) error {
	sourceNode := legacy.Path("source_node")

	uuid := sourceNode.Path("uuid").Data().(string)
	name := sourceNode.Path("name").Data().(string)

	legacy.DeleteP("source_node")
	legacy.SetP(uuid, "source_node.uuid")
	legacy.SetP(name, "source_node.name")

	// Internally-indexed docs of `type:shard` will set `shard.relocating_node` to `null`, if
	// the shard is not relocating. However, Metricbeat-indexed docs of `type:shard` will simply
	// not send the `shard.relocating_node` field if the shard is not relocating. So we normalize
	// by deleting the `shard.relocating_node` field from the internally-indexed doc if the shard
	// is not relocating.
	relocatingNodePath := "shard.relocating_node"
	if legacy.ExistsP(relocatingNodePath) {
		legacy.DeleteP(relocatingNodePath)
	}

	return nil
}
