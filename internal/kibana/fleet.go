// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/elastic/e2e-testing/internal/shell"
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
	fleetServer := shell.GetEnv("FLEET_URL", "fleet-server")
	fleetPort := 8220
	if fleetServer != "fleet-server" {
		u, err := url.Parse(fleetServer)
		host, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			log.Fatal("Could not determine fleet port from FLEET_URL")
		}
		fleetPort, _ = strconv.Atoi(port)
		fleetServer = host
	}

	cfg := &FleetConfig{
		EnrollmentToken:          token,
		ElasticsearchCredentials: "elastic:changeme",
		ElasticsearchPort:        9200,
		ElasticsearchURI:         "elasticsearch",
		KibanaPort:               5601,
		KibanaURI:                "kibana",
		FleetServerPort:          fleetPort,
		FleetServerURI:           fleetServer,
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
	flags := []string{
		"-e", "-v", "--force", "--insecure", "--enrollment-token=" + cfg.EnrollmentToken,
		"--url", fmt.Sprintf("http://%s:%d", cfg.FleetServerURI, cfg.FleetServerPort),
	}

	return flags
}
