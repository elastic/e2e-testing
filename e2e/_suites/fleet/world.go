// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"

	"github.com/elastic/e2e-testing/cli/services"
	"github.com/elastic/e2e-testing/e2e/steps"
)

// developerMode tears down the backend services (ES, Kibana, Package Registry)
// after a test suite. This is the desired behavior, but when developing, we maybe want to keep
// them running to speed up the development cycle.
// It can be overriden by the DEVELOPER_MODE env var
var developerMode = false

// ElasticAgentProcessName the name of the process for the Elastic Agent
const ElasticAgentProcessName = "elastic-agent"

// ElasticAgentServiceName the name of the service for the Elastic Agent
const ElasticAgentServiceName = "elastic-agent"

// FleetProfileName the name of the profile to run the runtime, backend services
const FleetProfileName = "fleet"

var agentVersionBase = "7.x-SNAPSHOT"

// agentVersion is the version of the agent to use
// It can be overriden by BEAT_VERSION env var
var agentVersion = agentVersionBase

// agentStaleVersion is the version of the agent to use as a base during upgrade
// It can be overriden by ELASTIC_AGENT_STALE_VERSION env var. Using latest GA as a default.
var agentStaleVersion = "7.11-SNAPSHOT"

// kibanaVersion is the version of the kibana to use
// It can be overriden by KIBANA_VERSION env var
var kibanaVersion = agentVersionBase

// stackVersion is the version of the stack to use
// It can be overriden by STACK_VERSION env var
var stackVersion = agentVersionBase

// profileEnv is the environment to be applied to any execution
// affecting the runtime dependencies (or profile)
var profileEnv map[string]string

// timeoutFactor a multiplier for the max timeout when doing backoff retries.
// It can be overriden by TIMEOUT_FACTOR env var
var timeoutFactor = 3

// All URLs running on localhost as Kibana is expected to be exposed there
const kibanaBaseURL = "http://localhost:5601"

var kibanaClient *services.KibanaClient

// IngestManagerTestSuite represents a test suite, holding references to the pieces needed to run the tests
type IngestManagerTestSuite struct {
	Fleet      *FleetTestSuite
	StandAlone *StandAloneTestSuite
}

func (imts *IngestManagerTestSuite) processStateOnTheHost(process string, state string) error {
	profile := FleetProfileName
	serviceName := ElasticAgentServiceName

	containerName := fmt.Sprintf("%s_%s_%s_%d", profile, imts.Fleet.Image+"-systemd", serviceName, 1)
	if imts.StandAlone.Hostname != "" {
		containerName = fmt.Sprintf("%s_%s_%d", profile, serviceName, 1)
	}

	return steps.CheckProcessStateOnTheHost(containerName, process, state, timeoutFactor)
}
