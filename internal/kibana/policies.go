// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Policy represents an Ingest Manager policy.
type Policy struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Namespace   string `json:"namespace"`
}

// ListPolicies returns the list of policies
func (c *Client) ListPolicies() ([]Policy, error) {
	statusCode, respBody, err := c.get(fmt.Sprintf("%s/agent_policies", FleetAPI))

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
	Vars    Vars          `json:"vars"`
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
func (c *Client) ListPackagePolicies() ([]PackageDataStream, error) {
	statusCode, respBody, err := c.get(fmt.Sprintf("%s/package_policies", FleetAPI))

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
