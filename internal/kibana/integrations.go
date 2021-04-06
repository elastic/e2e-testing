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
