// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
)

// FleetConfig represents the configuration for Fleet Server when building the enrollment command
type FleetConfig struct {
	ContainerName   string
	EnrollmentToken string
	// server
	ServerCredentials string
	ServerPolicyID    string
	ServerPort        int
}

// NewFleetConfig builds a new configuration for the fleet server agent, defaulting credentials and port
func NewFleetConfig(containerName string, token string) *FleetConfig {
	return &FleetConfig{
		ContainerName:     containerName,
		EnrollmentToken:   token,
		ServerCredentials: "elastic:changeme",
		ServerPort:        9200,
	}
}

func (cfg FleetConfig) flags() []string {
	if cfg.ServerPolicyID != "" {
		return []string{"--fleet-server", fmt.Sprintf("http://%s@%s:%d", cfg.ServerCredentials, cfg.ContainerName, cfg.ServerPort), "--enrollment-token", cfg.EnrollmentToken, "--fleet-server-policy", cfg.ServerPolicyID}
	}

	return []string{"--kibana-url=http://kibana:5601", "--enrollment-token=" + cfg.EnrollmentToken, "-f", "--insecure"}
}
