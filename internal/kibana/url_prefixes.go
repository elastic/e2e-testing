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
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

const (
	// BaseURL Kibana host address
	BaseURL = "http://localhost:5601"

	// FleetAPI is the prefix for all Kibana Fleet API resources.
	FleetAPI = "/api/fleet"

	// EndpointAPI is the endpoint API
	EndpointAPI = "/api/endpoint"
)

// Endpoint - Kibana endpoint information
type Endpoint struct {
	Scheme string
	Host   string
	Port   int
}

// GetKibanaEndpoint - capture kibana environment information for determining endpoint
func GetKibanaEndpoint() *Endpoint {
	remoteKibanaHost := shell.GetEnv("KIBANA_URL", "")
	if remoteKibanaHost != "" {
		remoteKibanaHost = utils.RemoveQuotes(remoteKibanaHost)
		u, err := url.Parse(remoteKibanaHost)
		if err != nil {
			log.WithFields(log.Fields{
				"url":   remoteKibanaHost,
				"error": err,
			}).Trace("Could not parse KIBANA_URL, will attempt with original.")
			return &Endpoint{
				Scheme: "http",
				Host:   "localhost",
				Port:   5601,
			}
		}
		host, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			log.Fatal("Could not determine host/port from KIBANA_URL")
		}
		kibanaPort, _ := strconv.Atoi(port)
		return &Endpoint{
			Scheme: u.Scheme,
			Host:   host,
			Port:   kibanaPort,
		}

	}
	return &Endpoint{
		Scheme: "http",
		Host:   "localhost",
		Port:   5601,
	}

}

// getBaseURL will pull in the baseurl or an alternative host based on settings
func getBaseURL() string {
	kibanaEndpoint := GetKibanaEndpoint()
	endpoint := fmt.Sprintf("%s://%s:%d", kibanaEndpoint.Scheme, kibanaEndpoint.Host, kibanaEndpoint.Port)
	return endpoint
}
