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

// stackVersion is the version of the stack to use
// It can be overriden by STACK_VERSION env var
var stackVersion = "8.0.0-SNAPSHOT"

// profileEnv is the environment to be applied to any execution
// affecting the runtime dependencies (or profile)
var profileEnv map[string]string

// queryRetryTimeout is the number of seconds between elasticsearch retry queries.
// It can be overriden by OP_RETRY_TIMEOUT env var
var queryRetryTimeout = 3

// All URLs running on localhost as Kibana is expected to be exposed there
const kibanaBaseURL = "http://localhost:5601"

func init() {
	config.Init()

	developerMode, _ = shell.GetEnvBool("DEVELOPER_MODE")
	if developerMode {
		log.Info("Running in Developer mode 💻: runtime dependencies between different test runs will be reused to speed up dev cycle")
	}

	queryRetryTimeout = shell.GetEnvInteger("OP_RETRY_TIMEOUT", queryRetryTimeout)
	stackVersion = shell.GetEnv("STACK_VERSION", stackVersion)
}

func IngestManagerFeatureContext(s *godog.Suite) {
	imts := IngestManagerTestSuite{
		Fleet: &FleetTestSuite{
			Installers: map[string]ElasticAgentInstaller{
				"centos-systemd": GetElasticAgentInstaller("centos-systemd"),
				"debian-systemd": GetElasticAgentInstaller("debian-systemd"),
			},
		},
		StandAlone: &StandAloneTestSuite{},
	}
	serviceManager := services.NewServiceManager()

	s.Step(`^the "([^"]*)" process is in the "([^"]*)" state on the host$`, imts.processStateOnTheHost)
	s.Step(`^the "([^"]*)" process is "([^"]*)" on the host$`, imts.processStateChangedOnTheHost)

	imts.Fleet.contributeSteps(s)
	imts.StandAlone.contributeSteps(s)

	s.BeforeSuite(func() {
		log.Debug("Installing ingest-manager runtime dependencies")

		workDir, _ := os.Getwd()
		profileEnv = map[string]string{
			"stackVersion":     stackVersion,
			"kibanaConfigPath": path.Join(workDir, "configurations", "kibana.config.yml"),
		}

		profile := "ingest-manager"
		err := serviceManager.RunCompose(true, []string{profile}, profileEnv)
		if err != nil {
			log.WithFields(log.Fields{
				"profile": profile,
			}).Fatal("Could not run the runtime dependencies for the profile.")
		}

		minutesToBeHealthy := 5 * time.Minute
		healthy, err := e2e.WaitForElasticsearch(minutesToBeHealthy)
		if !healthy {
			log.WithFields(log.Fields{
				"error":   err,
				"minutes": minutesToBeHealthy,
			}).Fatal("The Elasticsearch cluster could not get the healthy status")
		}

		healthyKibana, err := e2e.WaitForKibana(minutesToBeHealthy)
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
		log.Debug("Before Ingest Manager scenario")

		imts.StandAlone.Cleanup = false
	})
	s.AfterSuite(func() {
		if !developerMode {
			log.Debug("Destroying ingest-manager runtime dependencies")
			profile := "ingest-manager"

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
		log.Debug("After Ingest Manager scenario")

		if imts.StandAlone.Cleanup {
			serviceName := "elastic-agent"
			if !developerMode {
				_ = serviceManager.RemoveServicesFromCompose("ingest-manager", []string{serviceName}, profileEnv)
			} else {
				log.WithField("service", serviceName).Info("Because we are running in development mode, the service won't be stopped")
			}

			if _, err := os.Stat(imts.StandAlone.AgentConfigFilePath); err == nil {
				os.Remove(imts.StandAlone.AgentConfigFilePath)
				log.WithFields(log.Fields{
					"path": imts.StandAlone.AgentConfigFilePath,
				}).Debug("Elastic Agent configuration file removed.")
			}
		}

		if imts.Fleet.Cleanup {
			serviceName := imts.Fleet.Image
			if !developerMode {
				_ = serviceManager.RemoveServicesFromCompose("ingest-manager", []string{serviceName}, profileEnv)
			} else {
				log.WithField("service", serviceName).Info("Because we are running in development mode, the service won't be stopped")
			}

			err := imts.Fleet.removeToken()
			if err != nil {
				log.WithFields(log.Fields{
					"err":     err,
					"tokenID": imts.Fleet.CurrentTokenID,
				}).Warn("The enrollment token could not be deleted")
			}
		}
	})
}

// IngestManagerTestSuite represents a test suite, holding references to the pieces needed to run the tests
type IngestManagerTestSuite struct {
	Fleet      *FleetTestSuite
	StandAlone *StandAloneTestSuite
}

func (imts *IngestManagerTestSuite) processStateChangedOnTheHost(process string, state string) error {
	profile := "ingest-manager"
	image := "centos-systemd"
	serviceName := "centos-systemd"

	if state == "started" {
		return startAgent(profile, image, serviceName)
	} else if state != "stopped" {
		return godog.ErrPending
	}

	log.WithFields(log.Fields{
		"service": serviceName,
		"process": process,
	}).Debug("Stopping process on the service")

	stopCmds := []string{"pkill", "-9", process}
	if process == "elastic-agent" {
		stopCmds = []string{"systemctl", "stop", process}
	}

	err := execCommandInService(profile, image, serviceName, stopCmds, false)
	if err != nil {
		log.WithFields(log.Fields{
			"action":   state,
			"stopCmds": stopCmds,
			"error":    err,
			"service":  serviceName,
			"process":  process,
		}).Error("Could not stop process on the host")

		return err
	}

	// check process was stopped
	return imts.processStateOnTheHost(process, "stopped")
}

func (imts *IngestManagerTestSuite) processStateOnTheHost(process string, state string) error {
	// name of the container for the service:
	// we are using the Docker client instead of docker-compose
	// because it does not support returning the output of a
	// command: it simply returns error level
	serviceName := "ingest-manager_elastic-agent_1"
	timeout := 4 * time.Minute

	err := e2e.WaitForProcess(serviceName, process, state, timeout)
	if err != nil {
		if state == "started" {
			log.WithFields(log.Fields{
				"error":   err,
				"timeout": timeout,
			}).Error("The process was not found but should be present")
		} else {
			log.WithFields(log.Fields{
				"error":   err,
				"timeout": timeout,
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
	}).Debug("Retrieving container name from the Docker client")

	hostname, err := docker.ExecCommandIntoContainer(context.Background(), containerName, "root", []string{"hostname"})
	if err != nil {
		log.WithFields(log.Fields{
			"containerName": containerName,
			"error":         err,
		}).Error("Could not retrieve container name from the Docker client")
		return "", err
	}

	if strings.HasPrefix(hostname, "\x01\x00\x00\x00\x00\x00\x00\r") {
		hostname = strings.ReplaceAll(hostname, "\x01\x00\x00\x00\x00\x00\x00\r", "")
		log.WithFields(log.Fields{
			"hostname": hostname,
		}).Debug("Container name has been sanitized")
	}

	log.WithFields(log.Fields{
		"containerName": containerName,
		"hostname":      hostname,
	}).Info("Hostname retrieved from the Docker client")

	return hostname, nil
}
