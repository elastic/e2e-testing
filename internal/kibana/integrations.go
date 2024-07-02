package kibana

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm/v2"
)

// IntegrationPackage used to share information about a integration
type IntegrationPackage struct {
	ID      string `json:"-"`
	Name    string `json:"name"`
	Title   string `json:"title"`
	Version string `json:"version"`
}

// AddIntegrationToPolicy adds an integration to policy
func (c *Client) AddIntegrationToPolicy(ctx context.Context, packageDS PackageDataStream) error {
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	addIntegrationFn := func() error {
		span, _ := apm.StartSpanOptions(ctx, "Adding integration to policy", "fleet.package.add-to-policy", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		defer span.End()

		reqBody, err := json.Marshal(packageDS)
		if err != nil {
			log.WithFields(log.Fields{
				"elapsedTime": exp.GetElapsedTime(),
				"err":         err,
				"package":     packageDS,
				"retry":       retryCount,
			}).Warn("Could not convert policy-package (request) to JSON. Retrying")

			retryCount++

			return err
		}

		statusCode, respBody, err := c.post(ctx, fmt.Sprintf("%s/package_policies", FleetAPI), reqBody)
		if err != nil {
			log.WithFields(log.Fields{
				"elapsedTime": exp.GetElapsedTime(),
				"err":         err,
				"package":     packageDS,
				"body":        string(respBody),
				"retry":       retryCount,
			}).Warn("Could not add package to policy. Retrying")

			retryCount++

			return err
		}

		if statusCode != 200 {
			log.WithFields(log.Fields{
				"elapsedTime": exp.GetElapsedTime(),
				"err":         err,
				"statusCode":  statusCode,
				"response":    string(respBody),
				"package":     packageDS,
				"retry":       retryCount,
			}).Warn("could not add package to policy because of HTTP code is not 200")

			retryCount++
			return fmt.Errorf("could not add package to policy; API status code = %d; response body = %s", statusCode, respBody)
		}

		return nil
	}

	err := backoff.Retry(addIntegrationFn, exp)
	if err != nil {
		return err
	}

	return nil
}

// DeleteIntegrationFromPolicy adds an integration to policy
func (c *Client) DeleteIntegrationFromPolicy(ctx context.Context, packageDS PackageDataStream) error {
	span, _ := apm.StartSpanOptions(ctx, "Delete integration from policy", "fleet.integration.delete-from-policy", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	reqBody := `{"packagePolicyIds":["` + packageDS.ID + `"]}`
	statusCode, respBody, err := c.post(ctx, fmt.Sprintf("%s/package_policies/delete", FleetAPI), []byte(reqBody))
	if err != nil {
		return errors.Wrap(err, "could not delete integration from policy")
	}

	if statusCode != 200 {
		return fmt.Errorf("could not delete integration from policy; API status code = %d; response body = %s", statusCode, respBody)
	}
	return nil
}

// GetIntegrations returns all available integrations
func (c *Client) GetIntegrations(ctx context.Context) ([]IntegrationPackage, error) {
	span, _ := apm.StartSpanOptions(ctx, "Listing integrations", "fleet.integrations.items", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	statusCode, respBody, err := c.get(ctx, fmt.Sprintf("%s/epm/packages?experimental=true", FleetAPI))

	if err != nil {
		log.WithFields(log.Fields{
			"body":  string(respBody),
			"error": err,
		}).Error("Could not get Integration package")
		return []IntegrationPackage{}, err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"body":       string(respBody),
			"error":      err,
			"statusCode": statusCode,
		}).Error("Could not get Fleet's installed integrations")

		return nil, err
	}

	jsonParsed, err := gabs.ParseJSON(respBody)
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": jsonParsed,
		}).Error("Could not parse get response into JSON")
		return []IntegrationPackage{}, err
	}

	var resp struct {
		Packages []IntegrationPackage `json:"items"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return []IntegrationPackage{}, errors.Wrap(err, "Unable to convert integration package to JSON")
	}

	return resp.Packages, nil

}

// GetIntegrationByPackageName returns metadata from an integration from Fleet
func (c *Client) GetIntegrationByPackageName(ctx context.Context, packageName string) (IntegrationPackage, error) {
	span, _ := apm.StartSpanOptions(ctx, "Getting integration by package name", "fleet.integration.get-by-name", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("package", packageName)
	defer span.End()

	integrationPackages, err := c.GetIntegrations(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"package": packageName,
		}).Error("Could not get Integration packages list")
		return IntegrationPackage{}, err
	}

	for _, pkg := range integrationPackages {
		if strings.EqualFold(pkg.Name, packageName) || strings.EqualFold(pkg.Title, packageName) {
			return pkg, nil
		}
	}

	return IntegrationPackage{}, fmt.Errorf("unable to find package %s", packageName)
}

// GetIntegrationFromAgentPolicy get package policy from agent policy
func (c *Client) GetIntegrationFromAgentPolicy(ctx context.Context, packageName string, policy Policy) (PackageDataStream, error) {
	span, _ := apm.StartSpanOptions(ctx, "Getting integration from Elastic Agent policy", "fleet.integration.get-from-policy", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("package", packageName)
	defer span.End()

	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	foundPackageDataStream := PackageDataStream{}

	isPackageInPolicyFn := func() error {
		packagePolicies, err := c.ListPackagePolicies(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"elapsedTime": exp.GetElapsedTime(),
				"error":       err,
				"package":     packageName,
				"policyID":    policy.ID,
				"retries":     retryCount,
			}).Warn("Error retrieving the package policies")
			retryCount++
			return err
		}

		for _, child := range packagePolicies {
			if policy.ID != child.PolicyID {
				// not in the same policy: keep looping
				log.WithFields(log.Fields{
					"child.id":       child.ID,
					"child.PolicyID": child.PolicyID,
					"policy.id":      policy.ID,
					"retries":        retryCount,
				}).Trace("Policies differ on ID. Continuing")
				continue
			}

			// the package name coincides with policy's Name, policy's package title or policy's package name
			if strings.EqualFold(packageName, child.Name) || strings.EqualFold(packageName, child.Package.Title) || strings.EqualFold(packageName, child.Package.Name) {
				log.WithFields(log.Fields{
					"elapsedTime":        exp.GetElapsedTime(),
					"integrationPackage": packageName,
					"name":               child.Name,
					"packageName":        child.Package.Name,
					"packageTitle":       child.Package.Title,
					"packagePolicyID":    child.PolicyID,
					"policyID":           policy.ID,
					"retries":            retryCount,
				}).Trace("Package found in policy")

				foundPackageDataStream = child
				return nil
			}
		}

		log.WithFields(log.Fields{
			"elapsedTime":   exp.GetElapsedTime(),
			"package":       packageName,
			"policyID":      policy.ID,
			"policies":      packagePolicies,
			"policiesCount": len(packagePolicies),
			"retries":       retryCount,
		}).Warn("Package not found in policy")
		retryCount++
		return fmt.Errorf("package %s not found in policy %s", packageName, policy.ID)
	}

	err := backoff.Retry(isPackageInPolicyFn, exp)

	return foundPackageDataStream, err
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

// GetPackagePolicy sends a GET request to Fleet retrieving a package policy by its name
func (c *Client) GetPackagePolicy(ctx context.Context, name string) (PackageDataStream, error) {
	span, _ := apm.StartSpanOptions(ctx, "Retrieving package policy", "fleet.package-policy.get", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	statusCode, respBody, err := c.get(ctx, fmt.Sprintf("%s/package_policies/%s", FleetAPI, name))
	if err != nil {
		return PackageDataStream{}, errors.Wrap(err, "could not retrieve package policy")
	}

	if statusCode != 200 {
		return PackageDataStream{}, fmt.Errorf("could not retrieve package policy; API status code = %d; response body = %s", statusCode, respBody)
	}

	var item *ItemPackageDataStream
	if err := json.Unmarshal(respBody, &item); err != nil {
		return PackageDataStream{}, errors.Wrap(err, "Unable to convert package policy to JSON")
	}

	return item.PackageDS, nil
}

// GetMetadataFromSecurityApp sends a POST request to retrieve metadata from Security App
func (c *Client) GetMetadataFromSecurityApp(ctx context.Context) ([]SecurityEndpoint, error) {
	span, _ := apm.StartSpanOptions(ctx, "Getting metadata from Security App", "security.metadata.get", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	statusCode, respBody, err := c.get(ctx, fmt.Sprintf("%s/metadata", EndpointAPI))
	if err != nil {
		return []SecurityEndpoint{}, errors.Wrap(err, "could not get endpoint metadata")
	}

	jsonParsed, _ := gabs.ParseJSON(respBody)
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
func (c *Client) InstallIntegrationAssets(ctx context.Context, integration IntegrationPackage) (string, error) {
	span, _ := apm.StartSpanOptions(ctx, "Installing assets for integration", "fleet.package.install-assets", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	reqBody := `{}`
	statusCode, respBody, err := c.post(ctx, fmt.Sprintf("%s/epm/packages/%s/%s", FleetAPI, integration.Name, integration.Version), []byte(reqBody))
	if err != nil {
		return "", errors.Wrap(err, "could not install integration assets")
	}

	if statusCode != 200 {
		return "", fmt.Errorf("could not install integration assets; API status code = %d; response body = %s", statusCode, respBody)
	}

	var resp struct {
		Items struct {
			ID string `json:"id"`
		} `json:"items"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", errors.Wrap(err, "Unable to convert install integration assets to JSON")
	}

	return resp.Items.ID, nil
}

// IsAgentListedInSecurityApp retrieves the hosts from Endpoint to check if a hostname
// is listed in the Security App. For that, we will inspect the metadata, and will iterate
// through the hosts, until we get the proper hostname.
func (c *Client) IsAgentListedInSecurityApp(ctx context.Context, hostName string) (SecurityEndpoint, error) {
	span, _ := apm.StartSpanOptions(ctx, "Checking Elastic Agent in Security App", "security.elastic-agent.listed", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	hosts, err := c.GetMetadataFromSecurityApp(ctx)
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
func (c *Client) IsAgentListedInSecurityAppWithStatus(ctx context.Context, hostName string, desiredStatus string) (bool, error) {
	host, err := c.IsAgentListedInSecurityApp(ctx, hostName)
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
func (c *Client) IsPolicyResponseListedInSecurityApp(ctx context.Context, agentID string) (bool, error) {
	span, _ := apm.StartSpanOptions(ctx, "Checking if policy response is listed in the Security app", "security.policy-response.check", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	hosts, err := c.GetMetadataFromSecurityApp(ctx)
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
func (c *Client) UpdateIntegrationPackagePolicy(ctx context.Context, packageDS PackageDataStream) (string, error) {
	span, _ := apm.StartSpanOptions(ctx, "Updating integration package policy", "fleet.package-policy.update", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	// empty the ID as it won't be recoganized in the PUT body
	id := packageDS.ID
	packageDS.ID = ""
	reqBody, _ := json.Marshal(packageDS)
	statusCode, respBody, err := c.put(ctx, fmt.Sprintf("%s/package_policies/%s", FleetAPI, id), reqBody)
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
