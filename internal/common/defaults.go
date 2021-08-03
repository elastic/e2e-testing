// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package common

import (
	"strings"

	"github.com/elastic/e2e-testing/internal/deploy"
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

// ElasticAPMActive if APM is active in the test framework
var ElasticAPMActive = false

// FleetProfileName the name of the profile to run the runtime, backend services
const FleetProfileName = "fleet"

// FleetProfileServiceRequest a service request for the Fleet profile
var FleetProfileServiceRequest = deploy.NewServiceRequest(FleetProfileName)

// FleetServerAgentServiceName the name of the service for the Elastic Agent
const FleetServerAgentServiceName = "fleet-server"

// AgentStaleVersion is the version of the agent to use as a base during upgrade
// It can be overriden by ELASTIC_AGENT_STALE_VERSION env var. Using latest GA as a default.
var AgentStaleVersion = "7.13-SNAPSHOT"

// BeatVersionBase is the base version of the Beat to use
var BeatVersionBase = "8.0.0-SNAPSHOT"

// BeatVersion is the version of the Beat to use
// It can be overriden by BEAT_VERSION env var
var BeatVersion = BeatVersionBase

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

func init() {
	DeveloperMode = shell.GetEnvBool("DEVELOPER_MODE")
	if DeveloperMode {
		log.Info("Running in Developer mode 💻: runtime dependencies between different test runs will be reused to speed up dev cycle")
	}

	Provider = shell.GetEnv("PROVIDER", Provider)

	ElasticAPMActive = shell.GetEnvBool("ELASTIC_APM_ACTIVE")
	if ElasticAPMActive {
		log.WithFields(log.Fields{
			"apm-environment": shell.GetEnv("ELASTIC_APM_ENVIRONMENT", "local"),
		}).Info("Current execution will be instrumented 🛠")
	}
}

// ElasticVersion represents a version
type elasticVersion struct {
	Version         string // 8.0.0
	FullVersion     string // 8.0.0-abcdef-SNAPSHOT
	HashedVersion   string // 8.0.0-abcdef
	SnapshotVersion string // 8.0.0-SNAPSHOT
}

func newElasticVersion(version string) *elasticVersion {
	versionWithoutCommit := utils.RemoveCommitFromSnapshot(version)
	versionWithoutSnapshot := strings.ReplaceAll(version, "-SNAPSHOT", "")
	versionWithoutCommitAndSnapshot := strings.ReplaceAll(versionWithoutCommit, "-SNAPSHOT", "")

	ev := &elasticVersion{
		FullVersion:     version,
		HashedVersion:   versionWithoutSnapshot,
		SnapshotVersion: versionWithoutCommit,
		Version:         versionWithoutCommitAndSnapshot,
	}

	if utils.SnapshotHasCommit(version) {
		ev.HashedVersion = versionWithoutSnapshot
	}

	return ev
}

// GetCommitVersion returns a version including the version and the git commit
func GetCommitVersion(version string) string {
	return newElasticVersion(version).HashedVersion
}

// GetFullVersion returns a version including the full version: version, git commit and snapshot
func GetFullVersion(version string) string {
	return newElasticVersion(version).FullVersion
}

// GetSnapshotVersion returns a version including the snapshot version: version and snapshot
func GetSnapshotVersion(version string) string {
	return newElasticVersion(version).SnapshotVersion
}

// GetVersion returns a version without snapshot or commit
func GetVersion(version string) string {
	return newElasticVersion(version).Version
}

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
	fallbackVersion := BeatVersionBase
	if BeatVersion != BeatVersionBase {
		log.WithFields(log.Fields{
			"BeatVersionBase": BeatVersionBase,
			"BeatVersion":     BeatVersion,
		}).Trace("Beat Version provided: will be used as fallback")
		fallbackVersion = BeatVersion
	}
	BeatVersion = utils.CheckPRVersion(BeatVersion, fallbackVersion)

	StackVersion = shell.GetEnv("STACK_VERSION", BeatVersionBase)
	v, err = utils.GetElasticArtifactVersion(StackVersion)
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
		KibanaVersion, err = utils.GetElasticArtifactVersion(StackVersion)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"version": KibanaVersion,
			}).Fatal("Failed to get kibana version, aborting")
		}
	}

	log.WithFields(log.Fields{
		"BeatVersionBase": BeatVersionBase,
		"BeatVersion":     BeatVersion,
		"StackVersion":    StackVersion,
		"KibanaVersion":   KibanaVersion,
	}).Trace("Initial artifact versions defined")
}
