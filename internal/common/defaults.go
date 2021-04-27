// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package common

import "runtime"

// ElasticAgentServiceName the name of the service for the Elastic Agent
const ElasticAgentServiceName = "elastic-agent"

// FleetProfileName the name of the profile to run the runtime, backend services
const FleetProfileName = "fleet"

// FleetServerAgentServiceName the name of the service for the Elastic Agent
const FleetServerAgentServiceName = "fleet-server"

// AgentVersionBase is the base version of the agent to use
var AgentVersionBase = "8.0.0-SNAPSHOT"

// AgentVersion is the version of the agent to use
// It can be overriden by BEAT_VERSION env var
var AgentVersion = AgentVersionBase

// AgentStaleVersion is the version of the agent to use as a base during upgrade
// It can be overriden by ELASTIC_AGENT_STALE_VERSION env var. Using latest GA as a default.
var AgentStaleVersion = "7.11-SNAPSHOT"

// StackVersion is the version of the stack to use
// It can be overriden by STACK_VERSION env var
var StackVersion = AgentVersionBase

// KibanaVersion is the version of kibana to use
// It can be override by KIBANA_VERSION
var KibanaVersion = AgentVersionBase

// ProfileEnv is the environment to be applied to any execution
// affecting the runtime dependencies (or profile)
var ProfileEnv map[string]string

// GetElasticAgentProcessName returns the elastic agent process name
func GetElasticAgentProcessName() string {
	if runtime.GOOS == "windows" {
		return "elastic-agent.exe"
	}
	return "elastic-agent"
}
