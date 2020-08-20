package main

import (
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	curl "github.com/elastic/e2e-testing/cli/shell"
	log "github.com/sirupsen/logrus"
)

const ingestManagerIntegrationURL = kibanaBaseURL + "/api/ingest_manager/epm/packages/%s-%s"
const ingestManagerIntegrationsURL = kibanaBaseURL + "/api/ingest_manager/epm/packages?experimental=true&category="

// getIntegrationLatestVersion sends a GET request to Ingest Manager for the existing integrations
// checking if the desired integration exists in the package registry. If so, it will
// return name and version (latest) of the integration
func getIntegrationLatestVersion(integrationName string) (string, string, error) {
	r := createDefaultHTTPRequest(ingestManagerIntegrationsURL)
	body, err := curl.Get(r)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   ingestManagerIntegrationsURL,
		}).Error("Could not get Integrations")
		return "", "", err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return "", "", err
	}

	// data streams should contain array of elements
	integrations := jsonParsed.Path("response").Children()

	log.WithFields(log.Fields{
		"count": len(integrations),
	}).Debug("Integrations retrieved")

	for _, integration := range integrations {
		name := integration.Path("name").Data().(string)
		if name == strings.ToLower(integrationName) {
			version := integration.Path("version").Data().(string)
			return name, version, nil
		}
	}

	return "", "", fmt.Errorf("The %s integration was not found", integrationName)
}

// installIntegration sends a POST request to Ingest Manager installing the assets for an integration
func installIntegrationAssets(integration string, version string) error {
	url := fmt.Sprintf(ingestManagerIntegrationURL, integration, version)
	postReq := createDefaultHTTPRequest(url)

	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   url,
		}).Error("Could not install assets for the integration")
		return err
	}

	log.WithFields(log.Fields{
		"integration": integration,
		"version":     version,
	}).Debug("Assets for the integration where installed")

	return nil
}
