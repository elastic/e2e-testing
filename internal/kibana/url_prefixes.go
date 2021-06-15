// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"fmt"
	"net/url"

	"github.com/elastic/e2e-testing/internal/shell"
)

const (
	// BaseURL Kibana host address
	BaseURL = "http://localhost:5601"

	// FleetAPI is the prefix for all Kibana Fleet API resources.
	FleetAPI = "/api/fleet"

	// EndpointAPI is the endpoint API
	EndpointAPI = "/api/endpoint"
)

// getBaseURL will pull in the baseurl or an alternative host based on settings
func getBaseURL() string {
	// If a remote docker host is set we need to make sure that kibana is pointed there
	// since API calls happen outside of the docker network
	dockerHost := shell.GetEnv("DOCKER_HOST", "")
	if dockerHost != "" {
		u, err := url.Parse(dockerHost)
		if err != nil {
			return BaseURL
		}
		endpoint := fmt.Sprintf("http://%s:5601", u.Host)
		return endpoint
	}
	return BaseURL
}
