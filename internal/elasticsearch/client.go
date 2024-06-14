// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	curl "github.com/elastic/e2e-testing/internal/curl"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	es "github.com/elastic/go-elasticsearch/v8"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/module/apmelasticsearch/v2"
)

// Query a very reduced representation of an elasticsearch query, where
// we want to simply override the event.module and service.version fields
//
//nolint:unused
type Query struct {
	EventModule    string
	IndexName      string
	ServiceVersion string
}

// SearchResult wraps a search result
type SearchResult map[string]interface{}

// DeleteIndex deletes an index from the elasticsearch running in the host
func DeleteIndex(ctx context.Context, index string) error {
	span, _ := apm.StartSpanOptions(ctx, "Search", "elasticsearch.index.delete", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("index", index)
	defer span.End()

	esClient, err := getElasticsearchClient(ctx)
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

// Endpoint - Elastic search endpoint information
type Endpoint struct {
	Scheme      string
	Host        string
	Port        int
	Credentials string
}

// GetElasticSearchEndpoint - Query environment for correct endpoint information
func GetElasticSearchEndpoint() *Endpoint {
	creds := fmt.Sprintf("%s:%s", shell.GetEnv("ELASTICSEARCH_USERNAME", "elastic"), shell.GetEnv("ELASTICSEARCH_PASSWORD", "changeme"))
	remoteESHost := shell.GetEnv("ELASTICSEARCH_URL", "")
	if remoteESHost != "" {
		remoteESHost = utils.RemoveQuotes(remoteESHost)
		u, err := url.Parse(remoteESHost)
		if err != nil {
			log.WithField("error", err).Fatal("Could not parse ELASTICSEARCH_URL")
		}
		host, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			log.Fatal("Could not determine host/port from ELASTICSEARCH_URL")
		}
		remoteESHostPort, _ := strconv.Atoi(port)
		return &Endpoint{
			Scheme:      u.Scheme,
			Host:        host,
			Port:        remoteESHostPort,
			Credentials: creds,
		}
	}
	return &Endpoint{
		Scheme:      "http",
		Host:        "localhost",
		Port:        9200,
		Credentials: creds,
	}
}

// getElasticsearchClient returns a client connected to the running elasticseach, defined
// at configuration level. Then we will inspect the running container to get its port bindings
// and from them, get the one related to the Elasticsearch port (9200). As it is bound to a
// random port at localhost, we will build the URL with the bound port at localhost.
//
//nolint:unused
func getElasticsearchClient(ctx context.Context) (*es.Client, error) {
	esEndpoint := GetElasticSearchEndpoint()
	return getElasticsearchClientFromHostPort(ctx, esEndpoint.Host, esEndpoint.Port, esEndpoint.Scheme)
}

// getElasticsearchClientFromHostPort returns a client connected to a running elasticseach, defined
// at configuration level. Then we will inspect the running container to get its port bindings
// and from them, get the one related to the Elasticsearch port (9200). As it is bound to a
// random port at localhost, we will build the URL with the bound port at localhost.
//
//nolint:unused
func getElasticsearchClientFromHostPort(ctx context.Context, host string, port int, scheme string) (*es.Client, error) {
	cfg := es.Config{
		Addresses: []string{fmt.Sprintf("%s://%s:%d", scheme, host, port)},
		Username:  shell.GetEnv("ELASTICSEARCH_USERNAME", "admin"),
		Password:  shell.GetEnv("ELASTICSEARCH_PASSWORD", "changeme"),
	}

	// avoid using common properties to avoid cyclical references
	elasticAPMActive := shell.GetEnvBool("ELASTIC_APM_ACTIVE")
	if elasticAPMActive {
		cfg.Transport = apmelasticsearch.WrapRoundTripper(http.DefaultTransport)
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

// SecurityTokenResponse wraps a security token result
type SecurityTokenResponse struct {
	AccessToken    string `json:"access_token"`
	Authentication struct {
		Realm struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"authentication_realm"`
		Type        string `json:"authentication_type"`
		Email       string `json:"email"`
		Enabled     bool   `json:"enabled"`
		FullName    bool   `json:"full_name"`
		LookupRealm struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"lookup_realm"`
		Metadata struct {
			Reserved string `json:"_reserved"`
		} `json:"metadata"`
		Roles    []string `json:"roles"`
		Username string   `json:"username"`
	} `json:"authentication"`
	ExpiresIn int    `json:"expires_in"`
	Type      string `json:"type"`
}

// SecurityTokenRequest payload for requesting a security token
type SecurityTokenRequest struct {
	GrantType string `json:"grant_type"`
}

// GetAPIToken retrieves an OAuth API token
func GetAPIToken(ctx context.Context) (SecurityTokenResponse, error) {
	span, _ := apm.StartSpanOptions(ctx, "Get API Token", "elasticsearch.security.get-token", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	result := SecurityTokenResponse{}

	esClient, err := getElasticsearchClient(ctx)
	if err != nil {
		return result, err
	}

	securityToken := SecurityTokenRequest{
		GrantType: "client_credentials",
	}

	log.WithFields(log.Fields{
		"tokenRequest": securityToken,
	}).Debug("Retrieving Elasticsearch's API token")

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(securityToken)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error decoding Security Token struct")

		return result, err
	}

	res, err := esClient.Security.GetToken(&buf)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error retrieving security token from Elasticsearch")

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
			"error getting response from Elasticsearch. Status: %s, ResponseError: %v",
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
		"result": result,
	}).Debug("API Token retrieved from Elasticsearch")

	return result, nil
}

// Search provide search interface to ES
func Search(ctx context.Context, indexName string, query map[string]interface{}) (SearchResult, error) {
	span, _ := apm.StartSpanOptions(ctx, "Search", "elasticsearch.search", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("index", indexName)
	span.Context.SetLabel("query", query)
	defer span.End()

	result := SearchResult{}

	esClient, err := getElasticsearchClient(ctx)
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
	}).Trace("Elasticsearch query")

	res, err := esClient.Search(
		esClient.Search.WithIndex(indexName),
		esClient.Search.WithBody(&buf),
		esClient.Search.WithTrackTotalHits(true),
		esClient.Search.WithPretty(),
		esClient.Search.WithSize(10000),
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
			"error getting response from Elasticsearch. Status: %s, ResponseError: %v",
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

// WaitForElasticsearch waits for elasticsearch running to be healthy, returning false
// if elasticsearch does not get healthy status in a defined number of minutes.
func WaitForElasticsearch(ctx context.Context, maxTimeoutMinutes time.Duration) (bool, error) {
	exp := utils.GetExponentialBackOff(maxTimeoutMinutes)

	retryCount := 1

	clusterStatus := func() error {
		esClient, err := getElasticsearchClient(context.Background())
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("Could not obtain an Elasticsearch client")

			return err
		}

		span, _ := apm.StartSpanOptions(ctx, "Health", "elasticsearch.health", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		defer span.End()

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

// WaitForClusterHealth waits for the elasticsearch cluster to be healthy
func WaitForClusterHealth(ctx context.Context) error {
	span, _ := apm.StartSpanOptions(ctx, "ClusterHealth", "elasticsearch.cluster.health", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	esClient, err := getElasticsearchClient(ctx)
	if err != nil {
		return err
	}

	exp := utils.GetExponentialBackOff(60 * time.Second)

	retryCount := 1

	healthFunction := func() error {
		response, err := esClient.Cluster.Health()
		if err != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"retry":       retryCount,
				"elapsedTime": exp.GetElapsedTime(),
			}).Warn("The Elasticsearch Cluster Health API is not available yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"retries":     retryCount,
			"response":    response,
			"elapsedTime": exp.GetElapsedTime(),
		}).Trace("The Elasticsearch Cluster Health API is  available")

		if response.StatusCode != 200 {
			log.WithFields(log.Fields{
				"retries":     retryCount,
				"statusCode":  response.StatusCode,
				"elapsedTime": exp.GetElapsedTime(),
			}).Warn("The Elasticsearch Cluster is not healthy yet. Retrying")

			retryCount++

			return fmt.Errorf("the Elasticsearch Cluster is not healthy yet. Retrying")
		}

		return nil
	}

	return backoff.Retry(healthFunction, exp)
}

// WaitForIndices waits for the elasticsearch indices to return the list of indices.
func WaitForIndices() (string, error) {
	exp := utils.GetExponentialBackOff(60 * time.Second)

	esEndpoint := GetElasticSearchEndpoint()
	retryCount := 1
	body := ""

	catIndices := func() error {
		r := curl.HTTPRequest{
			URL:               fmt.Sprintf("%s://%s:%d/_cat/indices?v", esEndpoint.Scheme, esEndpoint.Host, esEndpoint.Port),
			BasicAuthPassword: shell.GetEnv("ELASTICSEARCH_PASSWORD", "changeme"),
			BasicAuthUser:     shell.GetEnv("ELASTICSEARCH_USERNAME", "admin"),
		}

		response, err := curl.Get(r)
		if err != nil {
			log.WithFields(log.Fields{
				"error":          err,
				"retry":          retryCount,
				"statusEndpoint": r.URL,
				"elapsedTime":    exp.GetElapsedTime(),
			}).Warn("The Elasticsearch Cat Indices API is not available yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"retries":        retryCount,
			"statusEndpoint": r.URL,
			"elapsedTime":    exp.GetElapsedTime(),
		}).Trace("The Elasticsearch Cat Indices API is available")

		body = response
		return nil
	}

	err := backoff.Retry(catIndices, exp)
	return body, err
}

// WaitForNumberOfHits waits for an elasticsearch query to return more than a number of hits,
// returning false if the query does not reach that number in a defined number of time.
func WaitForNumberOfHits(ctx context.Context, indexName string, query map[string]interface{}, desiredHits int, maxTimeout time.Duration) (SearchResult, error) {
	exp := utils.GetExponentialBackOff(maxTimeout)

	retryCount := 1
	result := SearchResult{}

	numberOfHits := func() error {
		hits, err := Search(ctx, indexName, query)
		if err != nil {
			log.WithFields(log.Fields{
				"desiredHits": desiredHits,
				"elapsedTime": exp.GetElapsedTime(),
				"error":       err,
				"index":       indexName,
				"retry":       retryCount,
			}).Warn("There was an error executing the query")

			retryCount++
			return err
		}

		hitsCount := len(hits["hits"].(map[string]interface{})["hits"].([]interface{}))
		if hitsCount < desiredHits {
			log.WithFields(log.Fields{
				"currentHits": hitsCount,
				"desiredHits": desiredHits,
				"elapsedTime": exp.GetElapsedTime(),
				"index":       indexName,
				"retry":       retryCount,
			}).Warn("Waiting for more hits in the index")

			retryCount++

			return fmt.Errorf("not enough hits in the %s index yet. Current: %d, Desired: %d", indexName, hitsCount, desiredHits)
		}

		result = hits

		log.WithFields(log.Fields{
			"currentHits": hitsCount,
			"desiredHits": desiredHits,
			"retries":     retryCount,
			"elapsedTime": exp.GetElapsedTime(),
		}).Info("Hits number satisfied")

		return nil
	}

	err := backoff.Retry(numberOfHits, exp)
	return result, err
}
