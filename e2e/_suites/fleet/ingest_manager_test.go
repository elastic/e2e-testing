// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/messages-go/v10"
	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/cli/docker"
	"github.com/elastic/e2e-testing/cli/services"
	"github.com/elastic/e2e-testing/cli/shell"
	"github.com/elastic/e2e-testing/e2e"
	"github.com/elastic/e2e-testing/e2e/steps"
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
	agentVersionBase = e2e.GetElasticArtifactVersion(agentVersionBase)

	timeoutFactor = shell.GetEnvInteger("TIMEOUT_FACTOR", timeoutFactor)
	agentVersion = shell.GetEnv("BEAT_VERSION", agentVersionBase)

	agentStaleVersion = shell.GetEnv("ELASTIC_AGENT_STALE_VERSION", agentStaleVersion)
	// check if stale version is an alias
	agentStaleVersion = e2e.GetElasticArtifactVersion(agentStaleVersion)

	useCISnapshots := shell.GetEnvBool("BEATS_USE_CI_SNAPSHOTS")
	if useCISnapshots && !strings.HasSuffix(agentStaleVersion, "-SNAPSHOT") {
		agentStaleVersion += "-SNAPSHOT"
	}

	// check if version is an alias
	agentVersion = e2e.GetElasticArtifactVersion(agentVersion)

	stackVersion = shell.GetEnv("STACK_VERSION", stackVersion)
	stackVersion = e2e.GetElasticArtifactVersion(stackVersion)

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
			"stackVersion":     stackVersion,
			"kibanaConfigPath": path.Join(workDir, "configurations", "kibana.config.yml"),
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
			agentPath := v.path
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

// we need the container name because we use the Docker Client instead of Docker Compose
func getContainerHostname(containerName string) (string, error) {
	log.WithFields(log.Fields{
		"containerName": containerName,
	}).Trace("Retrieving container name from the Docker client")

	hostname, err := docker.ExecCommandIntoContainer(context.Background(), containerName, "root", []string{"cat", "/etc/hostname"})
	if err != nil {
		log.WithFields(log.Fields{
			"containerName": containerName,
			"error":         err,
		}).Error("Could not retrieve container name from the Docker client")
		return "", err
	}

	log.WithFields(log.Fields{
		"containerName": containerName,
		"hostname":      hostname,
	}).Info("Hostname retrieved from the Docker client")

	return hostname, nil
}
