// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"

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
	// server
	ServerPolicyID string
}

// NewFleetConfig builds a new configuration for the fleet agent, defaulting ES credentials, URI and port.
// If the 'fleetServerMode' flag is true, the it will also retrieve the default policy ID for fleet server
func NewFleetConfig(token string, fleetServerMode bool) (*FleetConfig, error) {
	cfg := &FleetConfig{
		EnrollmentToken:          token,
		ElasticsearchCredentials: "elastic:changeme",
		ElasticsearchPort:        9200,
		ElasticsearchURI:         "elasticsearch",
		KibanaPort:               5601,
		KibanaURI:                "kibana",
	}

	if fleetServerMode {
		defaultFleetServerPolicy, err := getAgentDefaultPolicy("is_default_fleet_server")
		if err != nil {
			return nil, err
		}

		cfg.ServerPolicyID = defaultFleetServerPolicy.Path("id").Data().(string)

		log.WithFields(log.Fields{
			"elasticsearch":     cfg.ElasticsearchURI,
			"elasticsearchPort": cfg.ElasticsearchPort,
			"policyID":          cfg.ServerPolicyID,
			"token":             cfg.EnrollmentToken,
		}).Debug("Fleet Server config created")
	}

	return cfg, nil
}

func (cfg FleetConfig) flags() []string {
	baseFlags := []string{"-e", "-v", "--force", "--insecure", "--enrollment-token=" + cfg.EnrollmentToken}

	if cfg.ServerPolicyID != "" {
		baseFlags = append(baseFlags, "--fleet-server-insecure-http", "--fleet-server", fmt.Sprintf("http://%s@%s:%d", cfg.ElasticsearchCredentials, cfg.ElasticsearchURI, cfg.ElasticsearchPort), "--fleet-server-host=http://0.0.0.0", "--fleet-server-policy", cfg.ServerPolicyID)
	}

	return append(baseFlags, "--kibana-url", fmt.Sprintf("http://%s@%s:%d", cfg.ElasticsearchCredentials, cfg.KibanaURI, cfg.KibanaPort))
}

func (fts *FleetTestSuite) anAgentIsDeployedToFleetWithInstallerInFleetMode(image string, installerType string) error {
	fts.ElasticAgentStopped = true
	return fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(image, installerType, true)
}
