// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package common

// ElasticAgentProcessName the name of the process for the Elastic Agent
const ElasticAgentProcessName = "elastic-agent"

// ElasticAgentServiceName the name of the service for the Elastic Agent
const ElasticAgentServiceName = "elastic-agent"

// ElasticEndpointIntegrationTitle title for the Elastic Endpoint integration in the package registry.
// This value could change depending on the version of the package registry
// We are using the title because the feature files have to be super readable
// and the title is more readable than the name
const ElasticEndpointIntegrationTitle = "Endpoint Security"

// FleetProfileName the name of the profile to run the runtime, backend services
const FleetProfileName = "fleet"

// AgentVersionBase is the base version of the agent to use
var AgentVersionBase = "7.13.0-SNAPSHOT"

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
