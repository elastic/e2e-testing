// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/messages-go/v10"
	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/cli/services"
	"github.com/elastic/e2e-testing/cli/shell"
	"github.com/elastic/e2e-testing/e2e"
	log "github.com/sirupsen/logrus"
)

var imts IngestManagerTestSuite

func setUpSuite() {
	config.Init()

	kibanaClient = services.NewKibanaClient()

	developerMode = shell.GetEnvBool("DEVELOPER_MODE")
	if developerMode {
		log.Info("Running in Developer mode 💻: runtime dependencies between different test runs will be reused to speed up dev cycle")
	}

	// check if base version is an alias
	v, err := e2e.GetElasticArtifactVersion(agentVersionBase)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": agentVersionBase,
		}).Fatal("Failed to get agent base version, aborting")
	}
	agentVersionBase = v

	timeoutFactor = shell.GetEnvInteger("TIMEOUT_FACTOR", timeoutFactor)
	agentVersion = shell.GetEnv("BEAT_VERSION", agentVersionBase)

	// check if version is an alias
<<<<<<< HEAD
	agentVersion = e2e.GetElasticArtifactVersion(agentVersion)
	agentStaleVersion = shell.GetEnv("ELASTIC_AGENT_STALE_VERSION", agentStaleVersion)
=======
	v, err = e2e.GetElasticArtifactVersion(agentVersion)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": agentVersion,
		}).Fatal("Failed to get agent version, aborting")
	}
	agentVersion = v

>>>>>>> 00a568dc... fix: delay checking stale agent version until it's used (#1016)
	stackVersion = shell.GetEnv("STACK_VERSION", stackVersion)
	v, err = e2e.GetElasticArtifactVersion(stackVersion)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": stackVersion,
		}).Fatal("Failed to get stack version, aborting")
	}
	stackVersion = v

	kibanaVersion = shell.GetEnv("KIBANA_VERSION", "")
	if kibanaVersion == "" {
		// we want to deploy a released version for Kibana
		// if not set, let's use stackVersion
		kibanaVersion = stackVersion
	}

	imts = IngestManagerTestSuite{
		Fleet: &FleetTestSuite{
			Installers: map[string]ElasticAgentInstaller{}, // do not pre-initialise the map
		},
		StandAlone: &StandAloneTestSuite{},
	}
}

func InitializeIngestManagerTestScenario(ctx *godog.ScenarioContext) {
	ctx.BeforeScenario(func(*messages.Pickle) {
		log.Trace("Before Fleet scenario")

		imts.StandAlone.Cleanup = false

		imts.Fleet.beforeScenario()
	})

	ctx.AfterScenario(func(*messages.Pickle, error) {
		log.Trace("After Fleet scenario")

		if imts.StandAlone.Cleanup {
			imts.StandAlone.afterScenario()
		}

		if imts.Fleet.Cleanup {
			imts.Fleet.afterScenario()
		}
	})

	ctx.Step(`^the "([^"]*)" process is in the "([^"]*)" state on the host$`, imts.processStateOnTheHost)

	imts.Fleet.contributeSteps(ctx)
	imts.StandAlone.contributeSteps(ctx)
}

func InitializeIngestManagerTestSuite(ctx *godog.TestSuiteContext) {
	serviceManager := services.NewServiceManager()

	ctx.BeforeSuite(func() {
		setUpSuite()

		log.Trace("Installing Fleet runtime dependencies")

		workDir, _ := os.Getwd()
		profileEnv = map[string]string{
			"kibanaConfigPath": path.Join(workDir, "configurations", "kibana.config.yml"),
			"kibanaVersion":    kibanaVersion,
			"stackVersion":     stackVersion,
		}

		profileEnv["kibanaDockerNamespace"] = "kibana"
		if strings.HasPrefix(kibanaVersion, "pr") {
			// because it comes from a PR
			profileEnv["kibanaDockerNamespace"] = "observability-ci"
		}

		profile := FleetProfileName
		err := serviceManager.RunCompose(context.Background(), true, []string{profile}, profileEnv)
		if err != nil {
			log.WithFields(log.Fields{
				"profile": profile,
			}).Fatal("Could not run the runtime dependencies for the profile.")
		}

		minutesToBeHealthy := time.Duration(timeoutFactor) * time.Minute
		healthy, err := e2e.WaitForElasticsearch(context.Background(), minutesToBeHealthy)
		if !healthy {
			log.WithFields(log.Fields{
				"error":   err,
				"minutes": minutesToBeHealthy,
			}).Fatal("The Elasticsearch cluster could not get the healthy status")
		}

		healthyKibana, err := kibanaClient.WaitForKibana(context.Background(), minutesToBeHealthy)
		if !healthyKibana {
			log.WithFields(log.Fields{
				"error":   err,
				"minutes": minutesToBeHealthy,
			}).Fatal("The Kibana instance could not get the healthy status")
		}

		imts.Fleet.setup()

		imts.StandAlone.RuntimeDependenciesStartDate = time.Now().UTC()
	})

	ctx.AfterSuite(func() {
		if !developerMode {
			log.Debug("Destroying Fleet runtime dependencies")
			profile := FleetProfileName

			err := serviceManager.StopCompose(context.Background(), true, []string{profile})
			if err != nil {
				log.WithFields(log.Fields{
					"error":   err,
					"profile": profile,
				}).Warn("Could not destroy the runtime dependencies for the profile.")
			}
		}

		installers := imts.Fleet.Installers
		for k, v := range installers {
			agentPath := v.binaryPath
			if _, err := os.Stat(agentPath); err == nil {
				err = os.Remove(agentPath)
				if err != nil {
					log.WithFields(log.Fields{
						"err":       err,
						"installer": k,
						"path":      agentPath,
					}).Warn("Elastic Agent binary could not be removed.")
				} else {
					log.WithFields(log.Fields{
						"installer": k,
						"path":      agentPath,
					}).Debug("Elastic Agent binary was removed.")
				}
			}
		}
	})
}
