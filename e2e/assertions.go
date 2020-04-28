package e2e

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// assertHitsArePresent returns an error if no hits are present
//nolint:unused
func assertHitsArePresent(hits map[string]interface{}, q ElasticsearchQuery) error {
	hitsCount := len(hits["hits"].(map[string]interface{})["hits"].([]interface{}))
	if hitsCount == 0 {
		return fmt.Errorf(
			"There aren't documents for %s-%s on Metricbeat index %s",
			q.EventModule, q.ServiceVersion, q.IndexName)
	}

	return nil
}

// assertHitsDoNotContainErrors returns an error if any of the returned entries contains
// an "error.message" field in the "_source" document
//nolint:unused
func assertHitsDoNotContainErrors(hits map[string]interface{}, q ElasticsearchQuery) error {
	for _, hit := range hits["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		if val, ok := source.(map[string]interface{})["error"]; ok {
			if msg, exists := val.(map[string]interface{})["message"]; exists {
				log.WithFields(log.Fields{
					"ID":            hit.(map[string]interface{})["_id"],
					"error.message": msg,
				}).Error("Error Hit found")

				return fmt.Errorf(
					"There are errors for %s-%s on Metricbeat index %s",
					q.EventModule, q.ServiceVersion, q.IndexName)
			}
		}
	}

	return nil
}

// assertHitsEqualStructure returns an error if hits don't share structure
//nolint:unused
func assertHitsEqualStructure(legacyHits map[string]interface{}, metricbeatHits map[string]interface{}) error {
	legacyCount := len(legacyHits["hits"].(map[string]interface{})["hits"].([]interface{}))
	metricbeatCount := len(metricbeatHits["hits"].(map[string]interface{})["hits"].([]interface{}))
	if legacyCount != metricbeatCount {
		return fmt.Errorf(
			"There is a different number of documents for legacy (%d) and metricbeat (%d) monitoring",
			legacyCount, metricbeatCount)
	}

	return nil
}
