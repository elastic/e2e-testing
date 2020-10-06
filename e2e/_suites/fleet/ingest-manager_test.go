// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/messages-go/v10"
	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/cli/docker"
	"github.com/elastic/e2e-testing/cli/services"
	"github.com/elastic/e2e-testing/cli/shell"
	"github.com/elastic/e2e-testing/e2e"
	log "github.com/sirupsen/logrus"
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

// stackVersion is the version of the stack to use
// It can be overriden by STACK_VERSION env var
var stackVersion = "8.0.0-SNAPSHOT"

// profileEnv is the environment to be applied to any execution
// affecting the runtime dependencies (or profile)
var profileEnv map[string]string

// timeoutFactor a multiplier for the max timeout when doing backoff retries.
// It can be overriden by TIMEOUT_FACTOR env var
var timeoutFactor = 3

// All URLs running on localhost as Kibana is expected to be exposed there
const kibanaBaseURL = "http://localhost:5601"

var kibanaClient *services.KibanaClient

func init() {
	config.Init()

	kibanaClient = services.NewKibanaClient()

	developerMode, _ = shell.GetEnvBool("DEVELOPER_MODE")
	if developerMode {
		log.Info("Running in Developer mode 💻: runtime dependencies between different test runs will be reused to speed up dev cycle")
	}

	timeoutFactor = shell.GetEnvInteger("TIMEOUT_FACTOR", timeoutFactor)
	stackVersion = shell.GetEnv("STACK_VERSION", stackVersion)
}

func IngestManagerFeatureContext(s *godog.Suite) {
	imts := IngestManagerTestSuite{
		Fleet: &FleetTestSuite{
			Installers: map[string]ElasticAgentInstaller{
				"centos":         GetElasticAgentInstaller("centos"),
				"centos-systemd": GetElasticAgentInstaller("centos-systemd"),
				"debian":         GetElasticAgentInstaller("debian"),
				"debian-systemd": GetElasticAgentInstaller("debian-systemd"),
			},
		},
		StandAlone: &StandAloneTestSuite{},
	}
	serviceManager := services.NewServiceManager()

	s.Step(`^the "([^"]*)" process is in the "([^"]*)" state on the host$`, imts.processStateOnTheHost)

	imts.Fleet.contributeSteps(s)
	imts.StandAlone.contributeSteps(s)

	s.BeforeSuite(func() {
		log.Trace("Installing Fleet runtime dependencies")

		workDir, _ := os.Getwd()
		profileEnv = map[string]string{
			"stackVersion":     stackVersion,
			"kibanaConfigPath": path.Join(workDir, "configurations", "kibana.config.yml"),
		}

		profile := FleetProfileName
		err := serviceManager.RunCompose(true, []string{profile}, profileEnv)
		if err != nil {
			log.WithFields(log.Fields{
				"profile": profile,
			}).Fatal("Could not run the runtime dependencies for the profile.")
		}

		minutesToBeHealthy := time.Duration(timeoutFactor) * time.Minute
		healthy, err := e2e.WaitForElasticsearch(minutesToBeHealthy)
		if !healthy {
			log.WithFields(log.Fields{
				"error":   err,
				"minutes": minutesToBeHealthy,
			}).Fatal("The Elasticsearch cluster could not get the healthy status")
		}

		healthyKibana, err := kibanaClient.WaitForKibana(minutesToBeHealthy)
		if !healthyKibana {
			log.WithFields(log.Fields{
				"error":   err,
				"minutes": minutesToBeHealthy,
			}).Fatal("The Kibana instance could not get the healthy status")
		}

		imts.Fleet.setup()

		imts.StandAlone.RuntimeDependenciesStartDate = time.Now().UTC()
	})
	s.BeforeScenario(func(*messages.Pickle) {
		log.Trace("Before Fleet scenario")

		imts.StandAlone.Cleanup = false

		imts.Fleet.beforeScenario()
	})
	s.AfterSuite(func() {
		if !developerMode {
			log.Debug("Destroying Fleet runtime dependencies")
			profile := FleetProfileName

			err := serviceManager.StopCompose(true, []string{profile})
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
	s.AfterScenario(func(*messages.Pickle, error) {
		log.Trace("After Fleet scenario")

		if imts.StandAlone.Cleanup {
			imts.StandAlone.afterScenario()
		}

		if imts.Fleet.Cleanup {
			imts.Fleet.afterScenario()
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

	containerName := fmt.Sprintf("%s_%s_%s_%d", profile, imts.Fleet.Image, serviceName, 1)
	if imts.StandAlone.Hostname != "" {
		containerName = fmt.Sprintf("%s_%s_%d", profile, serviceName, 1)
	}

	return checkProcessStateOnTheHost(containerName, process, state)
}

// name of the container for the service:
// we are using the Docker client instead of docker-compose
// because it does not support returning the output of a
// command: it simply returns error level
func checkProcessStateOnTheHost(containerName string, process string, state string) error {
	timeout := time.Duration(timeoutFactor) * time.Minute

	err := e2e.WaitForProcess(containerName, process, state, timeout)
	if err != nil {
		if state == "started" {
			log.WithFields(log.Fields{
				"container ": containerName,
				"error":      err,
				"timeout":    timeout,
			}).Error("The process was not found but should be present")
		} else {
			log.WithFields(log.Fields{
				"container": containerName,
				"error":     err,
				"timeout":   timeout,
			}).Error("The process was found but shouldn't be present")
		}

		return err
	}

	return nil
}

func execCommandInService(profile string, image string, serviceName string, cmds []string, detach bool) error {
	serviceManager := services.NewServiceManager()

	composes := []string{
		profile, // profile name
		image,   // image for the service
	}
	composeArgs := []string{"exec", "-T"}
	if detach {
		composeArgs = append(composeArgs, "-d")
	}
	composeArgs = append(composeArgs, serviceName)
	composeArgs = append(composeArgs, cmds...)

	err := serviceManager.RunCommand(profile, composes, composeArgs, profileEnv)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmds,
			"error":   err,
			"service": serviceName,
		}).Error("Could not execute command in container")

		return err
	}

	return nil
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
