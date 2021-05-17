// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

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
	FleetServerPort          int
	FleetServerURI           string
}

// NewFleetConfig builds a new configuration for the fleet agent, defaulting fleet-server host, ES credentials, URI and port.
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

	log.WithFields(log.Fields{
		"elasticsearch":     cfg.ElasticsearchURI,
		"elasticsearchPort": cfg.ElasticsearchPort,
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

	flags := []string{
		"-e", "-v", "--force", "--insecure", "--enrollment-token=" + cfg.EnrollmentToken,
		"--url", fmt.Sprintf("http://%s:%d", cfg.FleetServerURI, cfg.FleetServerPort),
	}

	return flags
}
