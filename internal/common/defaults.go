// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package common

import (
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

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

<<<<<<< HEAD
// AgentVersionBase is the base version of the agent to use
var AgentVersionBase = "7.13.0-SNAPSHOT"
=======
// FleetServerAgentServiceName the name of the service for the Elastic Agent
const FleetServerAgentServiceName = "fleet-server"

// BeatVersionBase is the base version of the Beat to use
var BeatVersionBase = "8.0.0-SNAPSHOT"
>>>>>>> 4c3d3ebe... feat: simplify the initialisation of versions (#1159)

// BeatVersion is the version of the Beat to use
// It can be overriden by BEAT_VERSION env var
var BeatVersion = BeatVersionBase

// AgentStaleVersion is the version of the agent to use as a base during upgrade
// It can be overriden by ELASTIC_AGENT_STALE_VERSION env var. Using latest GA as a default.
var AgentStaleVersion = "7.11-SNAPSHOT"

// StackVersion is the version of the stack to use
// It can be overriden by STACK_VERSION env var
var StackVersion = BeatVersionBase

// KibanaVersion is the version of kibana to use
// It can be override by KIBANA_VERSION
var KibanaVersion = BeatVersionBase

// ProfileEnv is the environment to be applied to any execution
// affecting the runtime dependencies (or profile)
var ProfileEnv map[string]string
<<<<<<< HEAD
=======

// Provider is the deployment provider used, currently docker is supported
var Provider = "docker"

// InitVersions initialise default versions. We do not want to do it in the init phase
// supporting lazy-loading the versions when needed. Basically, the CLI part does not
// need to load them
func InitVersions() {
	v, err := utils.GetElasticArtifactVersion(BeatVersionBase)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": BeatVersionBase,
		}).Fatal("Failed to get Beat base version, aborting")
	}
	BeatVersionBase = v

	BeatVersion = shell.GetEnv("BEAT_VERSION", BeatVersionBase)

	// check if version is an alias
	v, err = utils.GetElasticArtifactVersion(BeatVersion)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": BeatVersion,
		}).Fatal("Failed to get Beat version, aborting")
	}
	BeatVersion = v

	// detects if the BeatVersion is set by the GITHUB_CHECK_SHA1 variable
	BeatVersion = utils.CheckPRVersion(BeatVersion, BeatVersionBase)

	StackVersion = shell.GetEnv("STACK_VERSION", BeatVersionBase)
	v, err = utils.GetElasticArtifactVersion(StackVersion)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": StackVersion,
		}).Fatal("Failed to get stack version, aborting")
	}
	StackVersion = v
}
>>>>>>> 4c3d3ebe... feat: simplify the initialisation of versions (#1159)
