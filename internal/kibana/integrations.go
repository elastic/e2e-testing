package kibana

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// IntegrationPackage used to share information about a integration
type IntegrationPackage struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name"`
	Title   string `json:"title"`
	Version string `json:"version"`
}

// AddIntegrationToPolicy adds an integration to policy
func (c *Client) AddIntegrationToPolicy(packageDS PackageDataStream) error {
	reqBody, err := json.Marshal(packageDS)
	if err != nil {
		return errors.Wrap(err, "could not convert policy-package (request) to JSON")
	}

	statusCode, respBody, err := c.post(fmt.Sprintf("%s/package_policies", FleetAPI), reqBody)
	if err != nil {
		return errors.Wrap(err, "could not add package to policy")
	}

	if statusCode != 200 {
		return fmt.Errorf("could not add package to policy; API status code = %d; response body = %s", statusCode, respBody)
	}
	return nil
}

// DeleteIntegrationFromPolicy adds an integration to policy
func (c *Client) DeleteIntegrationFromPolicy(packageDS PackageDataStream) error {
	reqBody := `{"packagePolicyIds":["` + packageDS.ID + `"]}`
	statusCode, respBody, err := c.post(fmt.Sprintf("%s/package_policies/delete", FleetAPI), []byte(reqBody))
	if err != nil {
		return errors.Wrap(err, "could not delete integration from policy")
	}

	if statusCode != 200 {
		return fmt.Errorf("could not delete integration from policy; API status code = %d; response body = %s", statusCode, respBody)
	}
	return nil
}

// GetIntegrations returns all available integrations
func (c *Client) GetIntegrations() ([]IntegrationPackage, error) {
	statusCode, respBody, err := c.get(fmt.Sprintf("%s/epm/packages", FleetAPI))

	if err != nil {
		log.WithFields(log.Fields{
			"body":  respBody,
			"error": err,
		}).Error("Could not get Integration package")
		return []IntegrationPackage{}, err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"body":       respBody,
			"error":      err,
			"statusCode": statusCode,
		}).Error("Could not get Fleet's installed integrations")

		return nil, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(respBody))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": jsonParsed,
		}).Error("Could not parse get response into JSON")
		return []IntegrationPackage{}, err
	}

	var resp struct {
		Packages []IntegrationPackage `json:"response"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return []IntegrationPackage{}, errors.Wrap(err, "Unable to convert integration package to JSON")
	}

	return resp.Packages, nil

}

// GetIntegrationByPackageName returns metadata from an integration from Fleet
func (c *Client) GetIntegrationByPackageName(packageName string) (IntegrationPackage, error) {
	integrationPackages, err := c.GetIntegrations()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Could not get Integration packages list")
		return IntegrationPackage{}, err
	}

	for _, pkg := range integrationPackages {
		if strings.EqualFold(pkg.Name, packageName) {
			return pkg, nil
		}
	}

	return IntegrationPackage{}, errors.New("Unable to find package")
}

// GetIntegrationFromAgentPolicy get package policy from agent policy
func (c *Client) GetIntegrationFromAgentPolicy(packageName string, policy Policy) (PackageDataStream, error) {
	packagePolicies, err := c.ListPackagePolicies()
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"policy": policy,
		}).Trace("An error retrieving the package policies")
		return PackageDataStream{}, err
	}

	for _, child := range packagePolicies {
		if policy.ID == child.PolicyID && strings.EqualFold(packageName, child.Name) {
			return child, nil
		}
	}

	return PackageDataStream{}, errors.New("Unable to find package in policy")
}

// SecurityEndpoint endpoint metadata
type SecurityEndpoint struct {
	Metadata struct {
		Status string `json:"host_status"`
		Host   struct {
			Hostname string `json:"hostname"`
			Name     string `json:"name"`
		} `json:"host"`
		Elastic struct {
			Agent struct {
				ID      string `json:"id"`
				Version string `json:"version"`
			} `json:"agent"`
		} `json:"elastic"`
		Endpoint struct {
			Policy struct {
				Applied struct {
					Name   string `json:"name"`
					Status string `json:"status"`
				} `json:"applied"`
			} `json:"policy"`
		} `json:"Endpoint"`
	} `json:"metadata"`
}

// GetMetadataFromSecurityApp sends a POST request to retrieve metadata from Security App
func (c *Client) GetMetadataFromSecurityApp() ([]SecurityEndpoint, error) {
	reqBody := `{}`
	statusCode, respBody, err := c.post(fmt.Sprintf("%s/metadata", EndpointAPI), []byte(reqBody))
	if err != nil {
		return []SecurityEndpoint{}, errors.Wrap(err, "could not get endpoint metadata")
	}

	jsonParsed, _ := gabs.ParseJSON([]byte(respBody))
	log.WithFields(log.Fields{
		"responseBody": jsonParsed,
	}).Trace("Endpoint Metadata Response")

	if statusCode != 200 {
		return []SecurityEndpoint{}, fmt.Errorf("could not get endpoint metadata; API status code = %d; response body = %s", statusCode, respBody)
	}

	var resp struct {
		Hosts []SecurityEndpoint `json:"hosts"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return []SecurityEndpoint{}, errors.Wrap(err, "Unable to convert metadata from security app to JSON")
	}

	return resp.Hosts, nil
}

// InstallIntegrationAssets sends a POST request to Fleet installing the assets for an integration
func (c *Client) InstallIntegrationAssets(integration IntegrationPackage) (string, error) {
	reqBody := `{}`
	statusCode, respBody, err := c.post(fmt.Sprintf("%s/epm/packages/%s-%s", FleetAPI, integration.Name, integration.Version), []byte(reqBody))
	if err != nil {
		return "", errors.Wrap(err, "could not install integration assets")
	}

	if statusCode != 200 {
		return "", fmt.Errorf("could not install integration assets; API status code = %d; response body = %s", statusCode, respBody)
	}

	var resp struct {
		Response struct {
			ID string `json:"id"`
		} `json:"response"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", errors.Wrap(err, "Unable to convert install integration assets to JSON")
	}

	return resp.Response.ID, nil
}

// IsAgentListedInSecurityApp retrieves the hosts from Endpoint to check if a hostname
// is listed in the Security App. For that, we will inspect the metadata, and will iterate
// through the hosts, until we get the proper hostname.
func (c *Client) IsAgentListedInSecurityApp(hostName string) (SecurityEndpoint, error) {
	hosts, err := c.GetMetadataFromSecurityApp()
	if err != nil {
		return SecurityEndpoint{}, err
	}

	for _, host := range hosts {
		metadataHostname := host.Metadata.Host.Hostname
		if metadataHostname == hostName {
			log.WithFields(log.Fields{
				"hostname": hostName,
			}).Debug("Hostname for the agent listed in the Security App")

			return host, nil
		}
	}

	return SecurityEndpoint{}, nil
}

// IsAgentListedInSecurityAppWithStatus inspects the metadata field for a hostname, obtained from
// the security App. We will check if the status matches the desired status, returning an error
// if the agent is not present in the Security App
func (c *Client) IsAgentListedInSecurityAppWithStatus(hostName string, desiredStatus string) (bool, error) {
	host, err := c.IsAgentListedInSecurityApp(hostName)
	if err != nil {
		log.WithFields(log.Fields{
			"hostname": hostName,
			"error":    err,
		}).Error("There was an error getting the agent in the Administration view in the Security app")
		return false, err
	}

	hostStatus := host.Metadata.Status
	log.WithFields(log.Fields{
		"desiredStatus": desiredStatus,
		"hostname":      hostName,
		"status":        hostStatus,
	}).Debug("Hostname for the agent listed with desired status in the Administration view in the Security App")

	return (hostStatus == desiredStatus), nil
}

// IsPolicyResponseListedInSecurityApp sends a POST request to Endpoint to check if a hostname
// is listed in the Security App. For that, we will inspect the metadata, and will iterate
// through the hosts, until we get the policy status, finally checking for the success
// status.
func (c *Client) IsPolicyResponseListedInSecurityApp(agentID string) (bool, error) {
	hosts, err := c.GetMetadataFromSecurityApp()
	if err != nil {
		return false, err
	}

	for _, host := range hosts {
		metadataAgentID := host.Metadata.Elastic.Agent.ID
		name := host.Metadata.Endpoint.Policy.Applied.Name
		status := host.Metadata.Endpoint.Policy.Applied.Status
		if metadataAgentID == agentID {
			log.WithFields(log.Fields{
				"agentID": agentID,
				"name":    name,
				"status":  status,
			}).Debug("Policy response for the agent listed in the Security App")

			return (status == "success"), nil
		}
	}

	return false, nil
}

// UpdateIntegrationPackagePolicy sends a PUT request to Fleet updating integration
// configuration
func (c *Client) UpdateIntegrationPackagePolicy(packageDS PackageDataStream) (string, error) {
	// empty the ID as it won't be recoganized in the PUT body
	id := packageDS.ID
	packageDS.ID = ""
	reqBody, _ := json.Marshal(packageDS)
	statusCode, respBody, err := c.put(fmt.Sprintf("%s/package_policies/%s", FleetAPI, id), reqBody)
	if err != nil {
		return "", errors.Wrap(err, "could not update integration package")
	}

	if statusCode != 200 {
		return "", fmt.Errorf("could not update package ; API status code = %d; response body = %s", statusCode, respBody)
	}
	var resp struct {
		Item struct {
			UpdatedAt string `json:"updated_at"`
		} `json:"item"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", errors.Wrap(err, "Unable to convert install updated package policy to JSON")
	}

	return resp.Item.UpdatedAt, nil
}
