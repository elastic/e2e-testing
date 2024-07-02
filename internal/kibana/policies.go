// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm/v2"
)

// FleetServicePolicy these values comes from the kibana.config.yml file at Fleet's profile dir
var FleetServicePolicy = Policy{
	ID:                   "fleet-server-policy",
	Name:                 "Fleet Server Policy",
	Description:          "Fleet Server policy",
	IsDefaultFleetServer: true,
}

// Policy represents an Ingest Manager policy.
type Policy struct {
	ID                   string `json:"id,omitempty"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	Namespace            string `json:"namespace"`
	IsDefault            bool   `json:"is_default"`
	IsManaged            bool   `json:"is_managed"`
	IsDefaultFleetServer bool   `json:"is_default_fleet_server"`
	AgentsCount          int    `json:"agents"` // Number of agents connected to Policy
	Status               string `json:"status"`
}

// GetDefaultPolicy gets the default policy or optionally the default fleet policy
// deprecated: will be removed in upcoming releases
func (c *Client) GetDefaultPolicy(ctx context.Context, fleetServer bool) (Policy, error) {
	span, _ := apm.StartSpanOptions(ctx, "Getting default policy", "fleet.package-policies.get-default", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	policies, err := c.ListPolicies(ctx)
	if err != nil {
		return Policy{}, err
	}

	for _, policy := range policies {
		if fleetServer && policy.IsDefaultFleetServer {
			log.WithField("policy", policy).Trace("Returning Default Fleet Server Policy")
			return policy, nil
		} else if !fleetServer && policy.IsDefault {
			log.WithField("policy", policy).Trace("Returning Default Agent Policy")
			return policy, nil
		}
	}
	return Policy{}, errors.New("Could not obtain default policy")
}

// ListPolicies returns the list of policies
func (c *Client) ListPolicies(ctx context.Context) ([]Policy, error) {
	span, _ := apm.StartSpanOptions(ctx, "Listing Elastic Agent policies", "fleet.agent-policies.list", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	statusCode, respBody, err := c.get(ctx, fmt.Sprintf("%s/agent_policies", FleetAPI))

	if err != nil {
		log.WithFields(log.Fields{
			"body":  string(respBody),
			"error": err,
		}).Error("Could not get Fleet's policies")
		return nil, err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"error":      err,
			"body":       string(respBody),
			"statusCode": statusCode,
		}).Error("Could not get Fleet's policies")

		return nil, err
	}

	var resp struct {
		Items []Policy `json:"items"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, "Unable to convert list of policies to JSON")
	}

	return resp.Items, nil
}

// DeleteAllPolicies deletes all policies except fleet_server and system
func (c *Client) DeleteAllPolicies(ctx context.Context) {
	span, _ := apm.StartSpanOptions(ctx, "Deleting all agent policy", "fleet.package-policies.delete", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	// Cleanup all package policies
	packagePolicies, err := c.ListPackagePolicies(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("The package policies could not be found")
	}
	for _, pkgPolicy := range packagePolicies {
		// Do not remove the fleet server package integration otherwise fleet server fails to bootstrap
		if !strings.Contains(pkgPolicy.Name, "fleet_server") && !strings.Contains(pkgPolicy.Name, "system") {
			log.WithField("pkgPolicy", pkgPolicy.Name).Trace("Removing package policy")
			err = c.DeleteIntegrationFromPolicy(ctx, pkgPolicy)
			if err != nil {
				log.WithFields(log.Fields{
					"err":           err,
					"packagePolicy": pkgPolicy,
				}).Error("The integration could not be deleted from the configuration")
			}
		}
	}
}

// CreatePolicy creates a new policy for agent to utilize
func (c *Client) CreatePolicy(ctx context.Context) (Policy, error) {
	span, _ := apm.StartSpanOptions(ctx, "Creating agent policy", "fleet.package-policies.create", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	policyUUID := uuid.New().String()

	reqBody := `{
		"description": "Test policy ` + policyUUID + `",
		"namespace": "default",
		"monitoring_enabled": ["logs", "metrics"],
		"name": "test-policy-` + policyUUID + `"
	}`

	statusCode, respBody, _ := c.post(ctx, fmt.Sprintf("%s/agent_policies", FleetAPI), []byte(reqBody))

	jsonParsed, err := gabs.ParseJSON(respBody)

	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": jsonParsed,
		}).Error("Could not parse get response into JSON")
		return Policy{}, err
	}

	log.WithFields(log.Fields{
		"status":   statusCode,
		"err":      err,
		"reqBody":  reqBody,
		"respBody": jsonParsed,
	}).Trace("Policy creation result")

	if statusCode != 200 {
		return Policy{}, fmt.Errorf("Could not create Fleet's policy, unhandled server error (%d)", statusCode)
	}

	if err != nil {
		return Policy{}, errors.Wrap(err, "Could not create Fleet's policy")
	}

	var resp struct {
		Item Policy `json:"item"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return Policy{}, errors.Wrap(err, "Unable to convert list of new policy to JSON")
	}

	return resp.Item, nil
}

// Var represents a single variable at the package or
// data stream level, encapsulating the data type of the
// variable and it's value.
type Var struct {
	Value interface{} `json:"value"`
	Type  string      `json:"type"`
}

// Vars is a collection of variables either at the package or
// data stream level.
type Vars map[string]Var

// DataStream represents a data stream within a package.
type DataStream struct {
	Type    string `json:"type"`
	Dataset string `json:"dataset"`
}

// Input represents a package-level input.
type Input struct {
	Type           string      `json:"type"`
	Enabled        bool        `json:"enabled"`
	Streams        []Stream    `json:"streams,omitempty"`
	Vars           Vars        `json:"vars,omitempty"`
	Config         interface{} `json:"config,omitempty"`
	CompiledStream interface{} `json:"compiled_stream,omitempty"`
}

// ItemPackageDataStream represents a single item for a package policy.
type ItemPackageDataStream struct {
	PackageDS PackageDataStream `json:"item"`
}

// PackageDataStream represents a request to add a single package's single data stream to a
// Policy in Ingest Manager.
type PackageDataStream struct {
	ID          string             `json:"id,omitempty"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Namespace   string             `json:"namespace"`
	PolicyID    string             `json:"policy_id"`
	Enabled     bool               `json:"enabled"`
	OutputID    string             `json:"output_id"`
	Inputs      []Input            `json:"inputs"`
	Package     IntegrationPackage `json:"package"`
}

// Stream represents a stream for an input
type Stream struct {
	DS      DataStream `json:"data_stream"`
	Enabled bool       `json:"enabled"`
	ID      string     `json:"id"`
	Vars    Vars       `json:"vars,omitempty"`
}

// ListPackagePolicies return list of package policies
func (c *Client) ListPackagePolicies(ctx context.Context) ([]PackageDataStream, error) {
	span, _ := apm.StartSpanOptions(ctx, "Listing package policies", "fleet.package-policies.items", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	statusCode, respBody, err := c.get(ctx, fmt.Sprintf("%s/package_policies", FleetAPI))

	if err != nil {
		log.WithFields(log.Fields{
			"body":  string(respBody),
			"error": err,
		}).Error("Could not get Fleet's package policies")
		return nil, err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"body":       string(respBody),
			"error":      err,
			"statusCode": statusCode,
		}).Error("Could not get Fleet's package policies")

		return nil, err
	}

	var resp struct {
		Items []PackageDataStream `json:"items"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, "Unable to convert list of package policies to JSON")
	}

	return resp.Items, nil
}
