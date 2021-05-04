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
<<<<<<< HEAD
	// server
	BootstrapFleetServer bool
	ServerPolicyID       string
=======
>>>>>>> 82380ced... chore: remove unused code (#1119)
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

<<<<<<< HEAD
	if !bootstrapFleetServer {
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
	}
=======
	log.WithFields(log.Fields{
		"elasticsearch":     cfg.ElasticsearchURI,
		"elasticsearchPort": cfg.ElasticsearchPort,
		"token":             cfg.EnrollmentToken,
	}).Debug("Fleet Server config created")
>>>>>>> 82380ced... chore: remove unused code (#1119)

	return cfg, nil
}

// Flags bootstrap flags for fleet server
func (cfg FleetConfig) Flags() []string {
<<<<<<< HEAD
	if cfg.BootstrapFleetServer {
		// TO-DO: remove all code to calculate the fleet-server policy, because it's inferred by the fleet-server
		return []string{
			"--force",
			"--fleet-server-es", fmt.Sprintf("http://%s@%s:%d", cfg.ElasticsearchCredentials, cfg.ElasticsearchURI, cfg.ElasticsearchPort),
		}
	}

=======
>>>>>>> 82380ced... chore: remove unused code (#1119)
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
