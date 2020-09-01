package main

import (
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	curl "github.com/elastic/e2e-testing/cli/shell"
	log "github.com/sirupsen/logrus"
)

const endpointMetadataURL = kibanaBaseURL + "/api/endpoint/metadata"
const ingestManagerIntegrationURL = kibanaBaseURL + "/api/ingest_manager/epm/packages/%s-%s"
const ingestManagerIntegrationDeleteURL = kibanaBaseURL + "/api/ingest_manager/package_configs/delete"
const ingestManagerIntegrationsURL = kibanaBaseURL + "/api/ingest_manager/epm/packages?experimental=true&category="
const ingestManagerIntegrationConfigsURL = kibanaBaseURL + "/api/ingest_manager/package_configs"
const ingestManagerIntegrationConfigURL = ingestManagerIntegrationConfigsURL + "/%s"

// IntegrationPackage used to share information about a integration
type IntegrationPackage struct {
	packageConfigID string          `json:"packageConfigId"`
	name            string          `json:"name"`
	title           string          `json:"title"`
	version         string          `json:"version"`
	json            *gabs.Container // json representation of the integration
}

// installIntegration sends a POST request to Ingest Manager adding an integration to a configuration
func addIntegrationToConfiguration(integrationPackage IntegrationPackage, configurationID string) (string, error) {
	postReq := createDefaultHTTPRequest(ingestManagerIntegrationConfigsURL)

	data := `{
		"name":"` + integrationPackage.name + `-test-name",
		"description":"` + integrationPackage.title + `-test-description",
		"namespace":"default",
		"config_id":"` + configurationID + `",
		"enabled":true,
		"output_id":"",
		"inputs":[],
		"package":{
			"name":"` + integrationPackage.name + `",
			"title":"` + integrationPackage.title + `",
			"version":"` + integrationPackage.version + `"
		}
	}`
	postReq.Payload = data

	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":    body,
			"error":   err,
			"url":     ingestManagerIntegrationConfigsURL,
			"payload": data,
		}).Error("Could not add integration to configuration")
		return "", err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return "", err
	}

	integrationConfigurationID := jsonParsed.Path("item.id").Data().(string)

	log.WithFields(log.Fields{
		"configurationID":            configurationID,
		"integrationConfigurationID": integrationConfigurationID,
		"integration":                integrationPackage.name,
		"version":                    integrationPackage.version,
	}).Info("Integration added to the configuration")

	return integrationConfigurationID, nil
}

// deleteIntegrationFromConfiguration sends a POST request to Ingest Manager deleting an integration from a configuration
func deleteIntegrationFromConfiguration(integrationPackage IntegrationPackage, configurationID string) error {
	postReq := createDefaultHTTPRequest(ingestManagerIntegrationDeleteURL)

	data := `{"packageConfigIds":["` + integrationPackage.packageConfigID + `"]}`
	postReq.Payload = data

	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":    body,
			"error":   err,
			"url":     ingestManagerIntegrationDeleteURL,
			"payload": data,
		}).Error("Could not delete integration from configuration")
		return err
	}

	log.WithFields(log.Fields{
		"configurationID": configurationID,
		"integration":     integrationPackage.name,
		"packageConfigId": integrationPackage.packageConfigID,
		"version":         integrationPackage.version,
	}).Info("Integration deleted from the configuration")

	return nil
}

// getIntegration returns metadata from an integration from Fleet, without the package ID
func getIntegration(packageName string, version string) (IntegrationPackage, error) {
	url := fmt.Sprintf(ingestManagerIntegrationURL, packageName, version)
	getReq := createDefaultHTTPRequest(url)

	body, err := curl.Get(getReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   url,
		}).Error("Could not get the integration from Package Registry")
		return IntegrationPackage{}, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse get response into JSON")
		return IntegrationPackage{}, err
	}

	response := jsonParsed.Path("response")
	integrationPackage := IntegrationPackage{
		name:    response.Path("name").Data().(string),
		title:   response.Path("title").Data().(string),
		version: response.Path("latestVersion").Data().(string),
	}

	return integrationPackage, nil
}

// getIntegrationFromAgentConfiguration inspects the integrations added to an agent configuration, returning the
// a struct representing the package, including the packageID for the integration in the configuration
func getIntegrationFromAgentConfiguration(packageName string, agentConfigID string) (IntegrationPackage, error) {
	url := fmt.Sprintf(ingestManagerAgentConfigURL, agentConfigID)
	reqReq := createDefaultHTTPRequest(url)

	body, err := curl.Get(reqReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":        body,
			"error":       err,
			"packageName": packageName,
			"configID":    agentConfigID,
			"url":         url,
		}).Error("Could not get integration packages from the configuration")
		return IntegrationPackage{}, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return IntegrationPackage{}, err
	}

	packageConfigs := jsonParsed.Path("item.package_configs").Children()
	for _, packageConfig := range packageConfigs {
		title := packageConfig.Path("package.title").Data().(string)
		if title == packageName {
			integrationPackage := IntegrationPackage{
				packageConfigID: packageConfig.Path("id").Data().(string),
				name:            packageConfig.Path("package.name").Data().(string),
				title:           title,
				version:         packageConfig.Path("package.version").Data().(string),
				json:            packageConfig,
			}

			log.WithFields(log.Fields{
				"package":  integrationPackage,
				"configID": agentConfigID,
			}).Debug("Package config found in the configuration")

			return integrationPackage, nil
		}
	}

	return IntegrationPackage{}, fmt.Errorf("%s package config not found in the configuration", packageName)
}

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
	}).Trace("Integrations retrieved")

	for _, integration := range integrations {
		title := integration.Path("title").Data().(string)
		if strings.ToLower(title) == strings.ToLower(integrationName) {
			name := integration.Path("name").Data().(string)
			version := integration.Path("version").Data().(string)
			log.WithFields(log.Fields{
				"name":    name,
				"title":   title,
				"version": version,
			}).Debug("Integration in latest version found")
			return name, version, nil
		}
	}

	return "", "", fmt.Errorf("The %s integration was not found", integrationName)
}

// installIntegration sends a POST request to Ingest Manager installing the assets for an integration
func installIntegrationAssets(integration string, version string) (IntegrationPackage, error) {
	url := fmt.Sprintf(ingestManagerIntegrationURL, integration, version)
	postReq := createDefaultHTTPRequest(url)

	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   url,
		}).Error("Could not install assets for the integration")
		return IntegrationPackage{}, err
	}

	log.WithFields(log.Fields{
		"integration": integration,
		"version":     version,
	}).Info("Assets for the integration where installed")

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse install response into JSON")
		return IntegrationPackage{}, err
	}
	response := jsonParsed.Path("response").Index(0)

	packageConfigID := response.Path("id").Data().(string)

	// get the integration again in the case it's already installed
	integrationPackage, err := getIntegration(integration, version)
	if err != nil {
		return IntegrationPackage{}, err
	}

	integrationPackage.packageConfigID = packageConfigID

	return integrationPackage, nil
}

// isAgentListedInSecurityApp sends a POST request to Endpoint to check if a hostname
// is listed in the Security App. For that, we will inspect the metadata, and will iterate
// through the hosts, until we get the proper hostname.
func isAgentListedInSecurityApp(hostName string) (*gabs.Container, error) {
	postReq := createDefaultHTTPRequest(endpointMetadataURL)
	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   postReq.URL,
		}).Error("Could not get endpoint metadata")
		return nil, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return nil, err
	}

	hosts := jsonParsed.Path("hosts").Children()

	log.WithFields(log.Fields{
		"hosts": hosts,
	}).Trace("Hosts in the Security App")

	for _, host := range hosts {
		metadataHostname := host.Path("metadata.host.hostname").Data().(string)
		if metadataHostname == hostName {
			log.WithFields(log.Fields{
				"hostname": hostName,
			}).Debug("Hostname for the agent listed in the Security App")

			return host, nil
		}
	}

	return nil, nil
}

// isAgentListedInSecurityAppWithStatus inspects the metadata field for a hostname, obtained from
// the security App. We will check if the status matches the desired status, returning an error
// if the agent is not present in the Security App
func isAgentListedInSecurityAppWithStatus(hostName string, desiredStatus string) (bool, error) {
	host, err := isAgentListedInSecurityApp(hostName)
	if err != nil {
		log.WithFields(log.Fields{
			"hostname": hostName,
			"error":    err,
		}).Error("There was an error getting the agent in the Security app")
		return false, err
	}

	if host == nil {
		return false, fmt.Errorf("The host %s is not listed in the Security App", hostName)
	}

	hostStatus := host.Path("host_status").Data().(string)
	log.WithFields(log.Fields{
		"desiredStatus": desiredStatus,
		"hostname":      hostName,
		"status":        hostStatus,
	}).Debug("Hostname for the agent listed with desired status in the Security App")

	return (hostStatus == desiredStatus), nil
}

// updateIntegrationPackageConfig sends a PUT request to Ingest Manager updating integration
// configuration
func updateIntegrationPackageConfig(packageConfigID string, payload string) (*gabs.Container, error) {
	url := fmt.Sprintf(ingestManagerIntegrationConfigURL, packageConfigID)
	putReq := createDefaultHTTPRequest(url)

	putReq.Payload = payload

	body, err := curl.Put(putReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   url,
		}).Error("Could not update integration configuration")
		return nil, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return nil, err
	}

	log.WithFields(log.Fields{
		"configID": packageConfigID,
	}).Debug("Configuration for the integration was updated")

	return jsonParsed, nil
}
