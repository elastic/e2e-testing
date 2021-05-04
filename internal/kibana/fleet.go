// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"fmt"

	"github.com/elastic/e2e-testing/internal/common"
	log "github.com/sirupsen/logrus"
)

// FleetConfig represents the configuration for Fleet Server when building the enrollment command
type FleetConfig struct {
	EnrollmentToken          string
	ElasticsearchPort        int
	ElasticsearchURI         string
	ElasticsearchCredentials string
	KibanaPort               int
	KibanaURI                string
	FleetServerPort          int
	FleetServerURI           string
	// server
	ServerPolicyID string
}

// NewFleetConfig builds a new configuration for the fleet agent, defaulting fleet-server host, ES credentials, URI and port.
// It will also retrieve the default policy ID for fleet server
func NewFleetConfig(token string) (*FleetConfig, error) {
	cfg := &FleetConfig{
		EnrollmentToken:          token,
		ElasticsearchCredentials: "elastic:changeme",
		ElasticsearchPort:        9200,
		ElasticsearchURI:         "elasticsearch",
		KibanaPort:               5601,
		KibanaURI:                "kibana",
		FleetServerPort:          8220,
		FleetServerURI:           "fleet-server",
	}

	client, err := NewClient()
	if err != nil {
		return cfg, err
	}

	defaultFleetServerPolicy, err := client.GetDefaultPolicy(true)
	if err != nil {
		return nil, err
	}

	cfg.ServerPolicyID = defaultFleetServerPolicy.ID

	log.WithFields(log.Fields{
		"elasticsearch":     cfg.ElasticsearchURI,
		"elasticsearchPort": cfg.ElasticsearchPort,
		"policyID":          cfg.ServerPolicyID,
		"token":             cfg.EnrollmentToken,
	}).Debug("Fleet Server config created")

	return cfg, nil
}

// Flags bootstrap flags for fleet server
func (cfg FleetConfig) Flags() []string {
	/*
		// agent using an already bootstrapped fleet-server
		fleetServerHost := "https://hostname_of_the_bootstrapped_fleet_server:8220"
		return []string{
			"-e", "-v", "--force", "--insecure",
			// ensure the enrollment belongs to the default policy
			"--enrollment-token=" + cfg.EnrollmentToken,
			"--url", fleetServerHost,
		}
	*/

	baseFlags := []string{"-e", "-v", "--force", "--insecure", "--enrollment-token=" + cfg.EnrollmentToken}
	if common.AgentVersionBase == "8.0.0-SNAPSHOT" {
		return append(baseFlags, "--url", fmt.Sprintf("https://%s@%s:%d", cfg.ElasticsearchCredentials, cfg.FleetServerURI, cfg.FleetServerPort))
	}

	if cfg.ServerPolicyID != "" {
		baseFlags = append(baseFlags, "--fleet-server-insecure-http", "--fleet-server", fmt.Sprintf("https://%s@%s:%d", cfg.ElasticsearchCredentials, cfg.ElasticsearchURI, cfg.ElasticsearchPort), "--fleet-server-host=http://0.0.0.0", "--fleet-server-policy", cfg.ServerPolicyID)
	}

	return append(baseFlags, "--kibana-url", fmt.Sprintf("http://%s@%s:%d", cfg.ElasticsearchCredentials, cfg.KibanaURI, cfg.KibanaPort))
}
