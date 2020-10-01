// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package services

import (
	"fmt"
	"strings"

	curl "github.com/elastic/e2e-testing/cli/shell"
	log "github.com/sirupsen/logrus"
)

// KibanaBaseURL All URLs running on localhost as Kibana is expected to be exposed there
const kibanaBaseURL = "http://localhost:5601"

const endpointMetadataURL = "/api/endpoint/metadata"

const ingestManagerAgentPoliciesURL = "/api/ingest_manager/agent_policies"
const ingestManagerAgentPolicyURL = ingestManagerAgentPoliciesURL + "/%s"

const ingestManagerIntegrationDeleteURL = "/api/ingest_manager/package_policies/delete"
const ingestManagerIntegrationPoliciesURL = "/api/ingest_manager/package_policies"
const ingestManagerIntegrationPolicyURL = ingestManagerIntegrationPoliciesURL + "/%s"

const ingestManagerIntegrationsURL = "/api/ingest_manager/epm/packages?experimental=true&category="
const ingestManagerIntegrationURL = "/api/ingest_manager/epm/packages/%s-%s"

// KibanaClient manages calls to Kibana APIs
type KibanaClient struct {
	baseURL string
	url     string
}

// NewKibanaClient returns a kibana client
func NewKibanaClient() *KibanaClient {
	return &KibanaClient{
		baseURL: kibanaBaseURL,
	}
}

func (k *KibanaClient) getURL() string {
	return k.baseURL + k.url
}

func (k *KibanaClient) withURL(path string) *KibanaClient {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	k.url = path

	return k
}

// AddIntegrationToPolicy sends a POST request to add an integration to a policy
func (k *KibanaClient) AddIntegrationToPolicy(packageName string, name string, title string, description string, version string, policyID string) (string, error) {
	payload := `{
		"name":"` + name + `",
		"description":"` + description + `",
		"namespace":"default",
		"policy_id":"` + policyID + `",
		"enabled":true,
		"output_id":"",
		"inputs":[],
		"package":{
			"name":"` + packageName + `",
			"title":"` + title + `",
			"version":"` + version + `"
		}
	}`

	k.withURL(ingestManagerIntegrationPoliciesURL)

	postReq := createDefaultHTTPRequest(k.getURL())
	postReq.Payload = payload

	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":    body,
			"error":   err,
			"url":     k.getURL(),
			"payload": payload,
		}).Error("Could not add integration to configuration")
		return "", err
	}

	return body, err
}

// DeleteIntegrationFromPolicy sends a POST request to delete an integration from policy
func (k *KibanaClient) DeleteIntegrationFromPolicy(packageConfigID string) (string, error) {
	payload := `{"packagePolicyIds":["` + packageConfigID + `"]}`

	k.withURL(ingestManagerIntegrationDeleteURL)

	postReq := createDefaultHTTPRequest(k.getURL())
	postReq.Payload = payload

	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":    body,
			"error":   err,
			"url":     k.getURL(),
			"payload": payload,
		}).Error("Could not delete integration from configuration")
		return "", err
	}

	return body, err
}

// GetIntegration sends a GET request to fetch an integration by name and version
func (k *KibanaClient) GetIntegration(packageName string, version string) (string, error) {
	k.withURL(fmt.Sprintf(ingestManagerIntegrationURL, packageName, version))

	getReq := createDefaultHTTPRequest(k.getURL())

	body, err := curl.Get(getReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   k.getURL(),
		}).Error("Could not get the integration from Package Registry")
		return "", err
	}

	return body, err
}

// GetIntegrationFromAgentPolicy sends a GET request to fetch an integration from a policy
func (k *KibanaClient) GetIntegrationFromAgentPolicy(agentPolicyID string) (string, error) {
	k.withURL(fmt.Sprintf(ingestManagerAgentPolicyURL, agentPolicyID))

	getReq := createDefaultHTTPRequest(k.getURL())

	body, err := curl.Get(getReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":     body,
			"error":    err,
			"policyID": agentPolicyID,
			"url":      k.getURL(),
		}).Error("Could not get integration packages from the policy")
		return "", err
	}

	return body, err
}

// GetIntegrations sends a GET request to fetch latest version for all installed integrations
func (k *KibanaClient) GetIntegrations() (string, error) {
	k.withURL(ingestManagerIntegrationsURL)

	getReq := createDefaultHTTPRequest(k.getURL())

	body, err := curl.Get(getReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   k.getURL(),
		}).Error("Could not get Integrations")
		return "", err
	}

	return body, err
}

// GetMetadataFromSecurityApp sends a POST request to retrieve metadata from Security App
func (k *KibanaClient) GetMetadataFromSecurityApp() (string, error) {
	k.withURL(endpointMetadataURL)

	postReq := createDefaultHTTPRequest(k.getURL())
	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   k.getURL(),
		}).Error("Could not get endpoint metadata")
		return "", err
	}

	return body, err
}

// InstallIntegrationAssets sends a POST request to Ingest Manager installing the assets for an integration
func (k *KibanaClient) InstallIntegrationAssets(integration string, version string) (string, error) {
	k.withURL(fmt.Sprintf(ingestManagerIntegrationURL, integration, version))

	postReq := createDefaultHTTPRequest(k.getURL())

	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   k.getURL(),
		}).Error("Could not install assets for the integration")
		return "", err
	}

	return body, err
}

// UpdateIntegrationPackageConfig sends a PUT request to Ingest Manager updating integration
// configuration
func (k *KibanaClient) UpdateIntegrationPackageConfig(packageConfigID string, payload string) (string, error) {
	k.withURL(fmt.Sprintf(ingestManagerIntegrationPolicyURL, packageConfigID))

	putReq := createDefaultHTTPRequest(k.getURL())
	putReq.Payload = payload

	body, err := curl.Put(putReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   k.getURL(),
		}).Error("Could not update integration configuration")
		return "", err
	}

	return body, err
}

// createDefaultHTTPRequest Creates a default HTTP request, including the basic auth,
// JSON content type header, and a specific header that is required by Kibana
func createDefaultHTTPRequest(url string) curl.HTTPRequest {
	return curl.HTTPRequest{
		BasicAuthUser:     "elastic",
		BasicAuthPassword: "changeme",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"kbn-xsrf":     "e2e-tests",
		},
		URL: url,
	}
}
