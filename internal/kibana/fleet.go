// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/elastic/e2e-testing/internal/elasticsearch"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
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
	FleetServerScheme        string
}

// NewFleetConfig builds a new configuration for the fleet agent, defaulting fleet-server host, ES credentials, URI and port.
func NewFleetConfig(token string) (*FleetConfig, error) {
	fleetServer := shell.GetEnv("FLEET_URL", "fleet-server")
	fleetPort := 8220
	fleetServerScheme := "http"
	if fleetServer != "fleet-server" {
		fleetServer = utils.RemoveQuotes(fleetServer)
		u, err := url.Parse(fleetServer)
		if err != nil {
			log.WithField("error", err).Fatal("Could not parse FLEET_URL")
		}
		host, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			log.Fatal("Could not determine fleet port from FLEET_URL")
		}
		fleetPort, _ = strconv.Atoi(port)
		fleetServer = host
		fleetServerScheme = u.Scheme
	}

	esEndpoint := elasticsearch.GetElasticSearchEndpoint()
	kbEndpoint := GetKibanaEndpoint()

	cfg := &FleetConfig{
		EnrollmentToken:          token,
		ElasticsearchCredentials: esEndpoint.Credentials,
		ElasticsearchPort:        esEndpoint.Port,
		ElasticsearchURI:         esEndpoint.Host,
		KibanaPort:               kbEndpoint.Port,
		KibanaURI:                kbEndpoint.Host,
		FleetServerPort:          fleetPort,
		FleetServerURI:           fleetServer,
		FleetServerScheme:        fleetServerScheme,
	}

	log.WithFields(log.Fields{
		"elasticsearch": fmt.Sprintf("%s:%d", cfg.ElasticsearchURI, cfg.ElasticsearchPort),
		"fleet-server":  fmt.Sprintf("%s:%d", cfg.FleetServerURI, cfg.FleetServerPort),
		"kibana":        fmt.Sprintf("%s:%d", cfg.KibanaURI, cfg.KibanaPort),
		"token":         cfg.EnrollmentToken,
	}).Debug("Fleet Server config created")

	return cfg, nil
}

// Flags bootstrap flags for fleet server
func (cfg FleetConfig) Flags() []string {
	flags := []string{
		"--e", "--force", "--insecure", "--enrollment-token=" + cfg.EnrollmentToken,
		"--url", cfg.FleetServerURL(),
	}

	return flags
}

// FleetServerURL returns the fleet-server URL in the config
func (cfg FleetConfig) FleetServerURL() string {
	return fmt.Sprintf("%s://%s:%d", cfg.FleetServerScheme, cfg.FleetServerURI, cfg.FleetServerPort)
}
