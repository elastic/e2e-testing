// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package e2e

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// AssertHitsArePresent returns an error if no hits are present
func AssertHitsArePresent(hits map[string]interface{}) error {
	if getHitsCount(hits) == 0 {
		return fmt.Errorf("There aren't documents in the index")
	}

	return nil
}

// AssertHitsAreNotPresent returns an error if hits are present
func AssertHitsAreNotPresent(hits map[string]interface{}) error {
	count := getHitsCount(hits)
	if count != 0 {
		return fmt.Errorf("There are %d documents in the index", count)
	}

	return nil
}

// AssertHitsDoNotContainErrors returns an error if any of the returned entries contains
// an "error.message" field in the "_source" document
func AssertHitsDoNotContainErrors(hits map[string]interface{}, q ElasticsearchQuery) error {
	errors := []interface{}{}

	iterableHits := hits["hits"].(map[string]interface{})["hits"].([]interface{})
	for _, hit := range iterableHits {
		source := hit.(map[string]interface{})["_source"]
		if val, ok := source.(map[string]interface{})["error"]; ok {
			if msg, exists := val.(map[string]interface{})["message"]; exists {
				log.WithFields(log.Fields{
					"ID":            hit.(map[string]interface{})["_id"],
					"error.message": msg,
				}).Error("Error Hit found")

				errors = append(errors, msg)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(
			"Errors where found for %s-%s on Metricbeat's %s index: %d error/s out of %d",
			q.EventModule, q.ServiceVersion, q.IndexName, len(errors), len(iterableHits))
	}

	return nil
}

func getHitsCount(hits map[string]interface{}) int {
	return len(hits["hits"].(map[string]interface{})["hits"].([]interface{}))
}
