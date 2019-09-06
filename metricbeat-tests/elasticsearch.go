package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/go-connections/nat"
	es "github.com/elastic/go-elasticsearch/v8"
	log "github.com/sirupsen/logrus"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
	"github.com/elastic/metricbeat-tests-poc/cli/docker"
)

// searchResult wraps a search result
type searchResult map[string]interface{}

// deleteIndex deletes an index from the elasticsearch of the stack
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
		"status":    res.Status,
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
		"status":     res.Status,
	}).Debug("Index Alias deleted using Elasticsearch Go client")

	return nil
}

// getElasticsearchClient returns a client connected to the running elasticseach, defined
// at configuration level. Then we will inspect the running container to get its port bindings
// and from them, get the one related to the Elasticsearch port (9200). As it is bound to a
// random port at localhost, we will build the URL with the bound port at localhost.
func getElasticsearchClient(stackName string) (*es.Client, error) {
	elasticsearchCfg, _ := config.GetServiceConfig("elasticsearch")
	elasticsearchCfg.Name = elasticsearchCfg.Name + "-" + stackName

	elasticsearchSrv := serviceManager.BuildFromConfig(elasticsearchCfg)

	esJSON, err := docker.InspectContainer(elasticsearchSrv.GetContainerName())
	if err != nil {
		log.WithFields(log.Fields{
			"containerName": elasticsearchSrv.GetContainerName(),
			"error":         err,
		}).Error("Could not inspect the elasticsearch container")

		return nil, err
	}

	ports := esJSON.NetworkSettings.Ports
	binding := ports[nat.Port("9200/tcp")]

	cfg := es.Config{
		Addresses: []string{fmt.Sprintf("http://localhost:%s", binding[0].HostPort)},
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
		}).Error("Error getting response from Elasticsearch")

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

		log.WithFields(log.Fields{
			"status": res.Status(),
			"type":   e["error"].(map[string]interface{})["type"],
			"reason": e["error"].(map[string]interface{})["reason"],
		}).Error("Error getting response from Elasticsearch")

		return result, fmt.Errorf(
			"Error getting response from Elasticsearch. Status: %s, Type: %s, Reason: %s",
			res.Status(),
			e["error"].(map[string]interface{})["type"],
			e["error"].(map[string]interface{})["reason"])
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
