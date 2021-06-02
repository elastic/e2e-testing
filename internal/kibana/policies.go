// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

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
			"body":  respBody,
			"error": err,
		}).Error("Could not get Fleet's policies")
		return nil, err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"error":      err,
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
	Type    string        `json:"type"`
	Enabled bool          `json:"enabled"`
	Streams []interface{} `json:"streams"`
	Vars    Vars          `json:"vars,omitempty"`
	Config  interface{}   `json:"config,omitempty"`
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

// ListPackagePolicies return list of package policies
func (c *Client) ListPackagePolicies(ctx context.Context) ([]PackageDataStream, error) {
	span, _ := apm.StartSpanOptions(ctx, "Listing package policies", "fleet.package-policies.list", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	statusCode, respBody, err := c.get(ctx, fmt.Sprintf("%s/package_policies", FleetAPI))

	if err != nil {
		log.WithFields(log.Fields{
			"body":  respBody,
			"error": err,
		}).Error("Could not get Fleet's package policies")
		return nil, err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
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
