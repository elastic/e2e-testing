// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

const (
	// BaseURL Kibana host address
	// FIXME: Must take into account a remote docker instance
	BaseURL = "http://localhost:5601"

	// FleetAPI is the prefix for all Kibana Fleet API resources.
	FleetAPI = "/api/fleet"

	// EndpointAPI is the endpoint API
	EndpointAPI = "/api/endpoint"
)
