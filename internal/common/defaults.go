// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package common

import (
	"path/filepath"

	"github.com/elastic/e2e-testing/internal/config"
	"github.com/elastic/e2e-testing/internal/io"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/pkg/downloads"
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

// ElasticAPMActive if APM is active in the test framework
var ElasticAPMActive = false

// FleetProfileName the name of the profile to run the runtime, backend services
const FleetProfileName = "fleet"

// FleetServerAgentServiceName the name of the service for the Elastic Agent
const FleetServerAgentServiceName = "fleet-server"

// BeatVersionBase is the base version of the Beat to use
var BeatVersionBase = "7.17.12-4a94a841-SNAPSHOT"

// BeatVersion is the version of the Beat to use
// It can be overriden by BEAT_VERSION env var
var BeatVersion = BeatVersionBase

// ElasticAgentVersion is the version of the Elastic Agent to use
// It can be overriden by ELASTIC_AGENT_VERSION env var
var ElasticAgentVersion = BeatVersionBase

// DeveloperMode if enabled will keep deployments around after test runs
var DeveloperMode = false

// KibanaVersion is the version of kibana to use
// It can be override by KIBANA_VERSION
var KibanaVersion = BeatVersionBase

// ProfileEnv is the environment to be applied to any execution
// affecting the runtime dependencies (or profile)
var ProfileEnv map[string]string

// Provider is the deployment provider used, currently docker is supported
var Provider = "docker"

// StackVersion is the version of the stack to use
// It can be overriden by STACK_VERSION env var
var StackVersion = BeatVersionBase

// elasticAgentWorkingDir is the working directory for temporary operations, such as
// downloading and extracting the agent
var elasticAgentWorkingDir string

func init() {
	config.Init()

	elasticAgentWorkingDir = filepath.Join(config.OpDir(), ElasticAgentServiceName)
	io.MkdirAll(elasticAgentWorkingDir)

	DeveloperMode = shell.GetEnvBool("DEVELOPER_MODE")
	if DeveloperMode {
		log.Info("Running in Developer mode 💻: runtime dependencies between different test runs will be reused to speed up dev cycle")
	}

	Provider = shell.GetEnv("PROVIDER", Provider)
	log.Infof("Provider is %s", Provider)

	ElasticAPMActive = shell.GetEnvBool("ELASTIC_APM_ACTIVE")
	if ElasticAPMActive {
		log.WithFields(log.Fields{
			"apm-environment": shell.GetEnv("ELASTIC_APM_ENVIRONMENT", "local"),
		}).Info("Current execution will be instrumented 🛠")
	}
}

// GetElasticAgentWorkingPath retrieve the path to the elastic-agent dir
// under the tool's working directory, at current user's home
func GetElasticAgentWorkingPath(paths ...string) string {
	elements := []string{elasticAgentWorkingDir}
	elements = append(elements, paths...)
	p := filepath.Join(elements...)

	// create dirs up to the last parent
	io.MkdirAll(filepath.Dir(p))

	return p
}

// InitVersions initialise default versions. We do not want to do it in the init phase
// supporting lazy-loading the versions when needed. Basically, the CLI part does not
// need to load them
func InitVersions() {
	v, err := downloads.GetElasticArtifactVersion(BeatVersionBase)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": BeatVersionBase,
		}).Fatal("Failed to get Beat base version, aborting")
	}
	BeatVersionBase = v

	BeatVersion = shell.GetEnv("BEAT_VERSION", BeatVersionBase)

	// check if version is an alias. For compatibility versions let's
	// support aliases in the format major.minor
	if downloads.IsAlias(BeatVersion) {
		v, err = downloads.GetElasticArtifactVersion(BeatVersion)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"version": BeatVersion,
			}).Fatal("Failed to get Beat version, aborting")
		}
		BeatVersion = v
	} else {
		log.WithFields(log.Fields{
			"version": BeatVersion,
		}).Trace("Version is not an alias.")
	}

	// detects if the BeatVersion is set by the GITHUB_CHECK_SHA1 variable
	fallbackVersion := BeatVersionBase
	if BeatVersion != BeatVersionBase {
		log.WithFields(log.Fields{
			"BeatVersionBase": BeatVersionBase,
			"BeatVersion":     BeatVersion,
		}).Trace("Beat Version provided: will be used as fallback")
		fallbackVersion = BeatVersion
	}
	BeatVersion = downloads.CheckPRVersion(BeatVersion, fallbackVersion)

	ElasticAgentVersion = shell.GetEnv("ELASTIC_AGENT_VERSION", BeatVersionBase)

	// check if version is an alias
	v, err = downloads.GetElasticArtifactVersion(ElasticAgentVersion)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": ElasticAgentVersion,
		}).Fatal("Failed to get Elastic Agent version, aborting")
	}
	ElasticAgentVersion = v

	// detects if the ElasticAgentVersion is set by the GITHUB_CHECK_SHA1 variable
	fallbackVersion = BeatVersionBase
	if ElasticAgentVersion != BeatVersionBase {
		log.WithFields(log.Fields{
			"BeatVersionBase":     BeatVersionBase,
			"ElasticAgentVersion": ElasticAgentVersion,
		}).Trace("Elastic Agent Version provided: will be used as fallback")
		fallbackVersion = ElasticAgentVersion
	}
	ElasticAgentVersion = downloads.CheckPRVersion(ElasticAgentVersion, fallbackVersion)

	StackVersion = shell.GetEnv("STACK_VERSION", BeatVersionBase)
	v, err = downloads.GetElasticArtifactVersion(StackVersion)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": StackVersion,
		}).Fatal("Failed to get stack version, aborting")
	}
	StackVersion = v

	KibanaVersion = shell.GetEnv("KIBANA_VERSION", "")
	if KibanaVersion == "" {
		// we want to deploy a released version for Kibana
		// if not set, let's use StackVersion
		KibanaVersion, err = downloads.GetElasticArtifactVersion(StackVersion)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"version": KibanaVersion,
			}).Fatal("Failed to get kibana version, aborting")
		}
	}

	downloads.GithubCommitSha1 = shell.GetEnv("GITHUB_CHECK_SHA1", "")
	downloads.GithubRepository = shell.GetEnv("GITHUB_CHECK_REPO", "elastic-agent")

	log.WithFields(log.Fields{
		"BeatVersionBase":     BeatVersionBase,
		"BeatVersion":         BeatVersion,
		"ElasticAgentVersion": ElasticAgentVersion,
		"GithubCommitSha":     downloads.GithubCommitSha1,
		"GithubRepository":    downloads.GithubRepository,
		"StackVersion":        StackVersion,
		"KibanaVersion":       KibanaVersion,
	}).Info("Initial artifact versions defined")
}
