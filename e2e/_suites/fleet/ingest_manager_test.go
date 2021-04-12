// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/messages-go/v10"
	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/cli/services"
	"github.com/elastic/e2e-testing/cli/shell"
	"github.com/elastic/e2e-testing/e2e"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/compose"
	"github.com/elastic/e2e-testing/internal/installer"
	log "github.com/sirupsen/logrus"
)

var imts IngestManagerTestSuite

func setUpSuite() {
	config.Init()

	kibanaClient, err := kibana.NewClient()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	if developerMode {
		log.Info("Running in Developer mode 💻: runtime dependencies between different test runs will be reused to speed up dev cycle")
	}

	// check if base version is an alias
	common.AgentVersionBase = utils.GetElasticArtifactVersion(common.AgentVersionBase)

	common.TimeoutFactor = shell.GetEnvInteger("TIMEOUT_FACTOR", common.TimeoutFactor)
	common.AgentVersion = shell.GetEnv("BEAT_VERSION", common.AgentVersionBase)

	common.AgentStaleVersion = shell.GetEnv("ELASTIC_AGENT_STALE_VERSION", common.AgentStaleVersion)
	// check if stale version is an alias
	common.AgentStaleVersion = utils.GetElasticArtifactVersion(common.AgentStaleVersion)

	useCISnapshots := shell.GetEnvBool("BEATS_USE_CI_SNAPSHOTS")
	if useCISnapshots && !strings.HasSuffix(common.AgentStaleVersion, "-SNAPSHOT") {
		common.AgentStaleVersion += "-SNAPSHOT"
	}

	// check if version is an alias
	common.AgentVersion = utils.GetElasticArtifactVersion(common.AgentVersion)

	common.StackVersion = shell.GetEnv("STACK_VERSION", common.StackVersion)
	common.StackVersion = utils.GetElasticArtifactVersion(common.StackVersion)

	imts = IngestManagerTestSuite{
		Fleet: &FleetTestSuite{
			kibanaClient: kibanaClient,
			Installers:   map[string]installer.ElasticAgentInstaller{}, // do not pre-initialise the map
		},
		StandAlone: &StandAloneTestSuite{
			kibanaClient: kibanaClient,
		},
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
	serviceManager := compose.NewServiceManager()

	ctx.BeforeSuite(func() {
		setUpSuite()

		log.Trace("Installing Fleet runtime dependencies")

		common.ProfileEnv = map[string]string{
			"stackVersion": common.StackVersion,
		}

		profile := common.FleetProfileName
		err := serviceManager.RunCompose(context.Background(), true, []string{profile}, common.ProfileEnv)
		if err != nil {
			log.WithFields(log.Fields{
				"profile": profile,
			}).Fatal("Could not run the runtime dependencies for the profile.")
		}

		minutesToBeHealthy := time.Duration(common.TimeoutFactor) * time.Minute
		healthy, err := elasticsearch.WaitForElasticsearch(context.Background(), minutesToBeHealthy)
		if !healthy {
			log.WithFields(log.Fields{
				"error":   err,
				"minutes": minutesToBeHealthy,
			}).Fatal("The Elasticsearch cluster could not get the healthy status")
		}

		kibanaClient, err := kibana.NewClient()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Unable to create kibana client")
		}

		healthyKibana, err := kibanaClient.WaitForReady(minutesToBeHealthy)
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
			profile := common.FleetProfileName

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
			agentPath := v.BinaryPath
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
