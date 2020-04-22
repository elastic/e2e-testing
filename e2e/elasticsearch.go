package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	es "github.com/elastic/go-elasticsearch/v8"
	log "github.com/sirupsen/logrus"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
)

// ElasticsearchQuery a very reduced representation of an elasticsearch query, where
// we want to simply override the event.module and service.version fields
//nolint:unused
type ElasticsearchQuery struct {
	EventModule    string
	IndexName      string
	ServiceVersion string
}

// searchResult wraps a search result
//nolint:unused
type searchResult map[string]interface{}

// deleteIndex deletes an index from the elasticsearch of the stack
//nolint:unused
func deleteIndex(ctx context.Context, stackName string, index string) error {
	esClient, err := getElasticsearchClient(stackName)
	if err != nil {
		return err
	}

	res, err := esClient.Indices.Delete([]string{index})
	if err != nil {
		log.WithFields(log.Fields{
			"indexName": index,
			"error":     err,
		}).Error("Could not delete index using Elasticsearch Go client")

		return err
	}
	log.WithFields(log.Fields{
		"indexName": index,
		"status":    res.Status(),
	}).Debug("Index deleted using Elasticsearch Go client")

	res, err = esClient.Indices.DeleteAlias([]string{index}, []string{index})
	if err != nil {
		log.WithFields(log.Fields{
			"indexAlias": index,
			"error":      err,
		}).Error("Could not delete index alias using Elasticsearch Go client")

		return err
	}
	log.WithFields(log.Fields{
		"indexAlias": index,
		"status":     res.Status(),
	}).Debug("Index Alias deleted using Elasticsearch Go client")

	return nil
}

// getElasticsearchClient returns a client connected to the running elasticseach, defined
// at configuration level. Then we will inspect the running container to get its port bindings
// and from them, get the one related to the Elasticsearch port (9200). As it is bound to a
// random port at localhost, we will build the URL with the bound port at localhost.
//nolint:unused
func getElasticsearchClient(stackName string) (*es.Client, error) {
	elasticsearchCfg, _ := config.GetServiceConfig("elasticsearch")
	elasticsearchCfg.Name = elasticsearchCfg.Name + "-" + stackName

	cfg := es.Config{
		Addresses: []string{"http://localhost:9200"},
	}
	esClient, err := es.NewClient(cfg)
	if err != nil {
		log.WithFields(log.Fields{
			"config": cfg,
			"error":  err,
		}).Error("Could not obtain an Elasticsearch client")

		return nil, err
	}

	return esClient, nil
}

// maxAttempts could be redefined in the OP_QUERY_MAX_ATTEMPTS environment variable
//nolint:unused
func retrySearch(stackName string, indexName string, esQuery map[string]interface{}, maxAttempts int, retryTimeout int) (searchResult, error) {
	totalRetryTime := maxAttempts * retryTimeout

	for attempt := maxAttempts; attempt > 0; attempt-- {
		result, err := search(stackName, indexName, esQuery)
		if err == nil {
			return result, nil
		}

		if attempt > 1 {
			log.WithFields(log.Fields{
				"attempt":       attempt,
				"errorCause":    err.Error(),
				"index":         indexName,
				"query":         esQuery,
				"retryAttempts": maxAttempts,
				"retryTimeout":  retryTimeout,
			}).Debugf("Waiting %d seconds for the index to be ready", retryTimeout)
			time.Sleep(time.Duration(retryTimeout) * time.Second)
		}
	}

	err := fmt.Errorf("Could not send query to Elasticsearch in the specified time (%d seconds)", totalRetryTime)

	log.WithFields(log.Fields{
		"error":         err,
		"query":         esQuery,
		"retryAttempts": maxAttempts,
		"retryTimeout":  retryTimeout,
	}).Error(err.Error())

	return searchResult{}, err
}

//nolint:unused
func search(stackName string, indexName string, query map[string]interface{}) (searchResult, error) {
	result := searchResult{}

	esClient, err := getElasticsearchClient(stackName)
	if err != nil {
		return result, err
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error encoding Elasticsearch query")

		return result, err
	}

	log.WithFields(log.Fields{
		"index": indexName,
		"query": fmt.Sprintf("%s", query),
	}).Debug("Elasticsearch query")

	res, err := esClient.Search(
		esClient.Search.WithIndex(indexName),
		esClient.Search.WithBody(&buf),
		esClient.Search.WithTrackTotalHits(true),
		esClient.Search.WithPretty(),
	)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error performing search on Elasticsearch")

		return result, err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Error parsing error response body from Elasticsearch")

			return result, err
		}

		err := fmt.Errorf(
			"Error getting response from Elasticsearch. Status: %s, ResponseError: %v",
			res.Status(), e)

		return result, err
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error parsing response body from Elasticsearch")

		return result, err
	}

	log.WithFields(log.Fields{
		"status": res.Status(),
		"hits":   int(result["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64)),
		"took":   int(result["took"].(float64)),
	}).Debug("Response information")

	return result, nil
}

// waitForElasticsearchHealthy waits for elasticsearch to be healthy, returning false
// if elasticsearch does not get healthy status in a defined number of minutes.
//nolint:unused
func waitForElasticsearch(maxTimeoutMinutes time.Duration, stackName string) (bool, error) {
	var (
		initialInterval     = 500 * time.Millisecond
		randomizationFactor = 0.5
		multiplier          = 2.0
		maxInterval         = 5 * time.Second
		maxElapsedTime      = maxTimeoutMinutes * time.Minute
	)

	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = initialInterval
	exp.RandomizationFactor = randomizationFactor
	exp.Multiplier = multiplier
	exp.MaxInterval = maxInterval
	exp.MaxElapsedTime = maxElapsedTime

	retryCount := 1

	clusterStatus := func() error {
		esClient, err := getElasticsearchClient(stackName)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("Could not obtain an Elasticsearch client")

			return err
		}

		if _, err := esClient.Cluster.Health(); err != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"retry":       retryCount,
				"elapsedTime": exp.GetElapsedTime(),
			}).Warn("The Elasticsearch cluster is not healthy yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"retries":     retryCount,
			"elapsedTime": exp.GetElapsedTime(),
		}).Info("The Elasticsearch cluster is healthy")

		return nil
	}

	err := backoff.Retry(clusterStatus, exp)
	if err != nil {
		return false, err
	}

	return true, nil
}
