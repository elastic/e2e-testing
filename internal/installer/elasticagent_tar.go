// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"fmt"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

// elasticAgentTARPackage implements operations for a RPM installer
type elasticAgentTARPackage struct {
	service  string
	executor func(service string, cmd []string) (string, error)
	copier   func(service string, files []string) error
}

// NewElasticAgentTARPackage creates an instance for the RPM installer
func NewElasticAgentTARPackage(service string, executor func(service string, cmd []string) (string, error), copier func(service string, files []string) error) Package {
	return &elasticAgentTARPackage{
		service:  service,
		executor: executor,
		copier:   copier,
	}
}

// AddFiles will add files into the service environment, default destination is /
func (i *elasticAgentTARPackage) AddFiles(files []string) error {
	return i.copier(i.service, files)
}

// Install installs a TAR package
func (i *elasticAgentTARPackage) Install() error {
	log.Trace("No TAR install instructions")
	return nil
}

// Exec will execute a command within the service environment
func (i *elasticAgentTARPackage) Exec(args []string) (string, error) {
	output, err := i.executor(i.service, args)
	return output, err
}

// Enroll will enroll the agent into fleet
func (i *elasticAgentTARPackage) Enroll(token string) error {

	cfg, _ := kibana.NewFleetConfig(token)
	args := []string{"/elastic-agent/elastic-agent", "install"}
	for _, arg := range cfg.Flags() {
		args = append(args, arg)
	}

	output, err := i.Exec(args)
	log.Trace(output)
	if err != nil {
		return fmt.Errorf("Failed to install the agent with subcommand: %v", err)
	}
	return nil
}

// InstallCerts installs the certificates for a TAR package, using the right OS package manager
func (i *elasticAgentTARPackage) InstallCerts() error {
	return nil
}

// Logs prints logs of service
func (i *elasticAgentTARPackage) Logs() error {
	return nil
}

// Postinstall executes operations after installing a TAR package
func (i *elasticAgentTARPackage) Postinstall() error {
	return nil
}

// Preinstall executes operations before installing a TAR package
func (i *elasticAgentTARPackage) Preinstall() error {
	artifact := "elastic-agent"
	os := "linux"
	arch := "x86_64"
	extension := "tar.gz"

	binaryName := utils.BuildArtifactName(artifact, common.AgentVersion, common.AgentVersionBase, os, arch, extension, false)
	binaryPath, err := utils.FetchBeatsBinary(binaryName, artifact, common.AgentVersion, common.AgentVersionBase, common.TimeoutFactor, true)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":  artifact,
			"version":   common.AgentVersion,
			"os":        os,
			"arch":      arch,
			"extension": extension,
			"error":     err,
		}).Error("Could not download the binary for the agent")
		return err
	}

	err = i.AddFiles([]string{binaryPath})
	if err != nil {
		return err
	}

	output, _ := i.Exec([]string{"mv", fmt.Sprintf("/%s-%s-%s-%s", artifact, common.AgentVersion, os, arch), "/elastic-agent"})
	log.WithField("output", output).Trace("Moved elastic-agent")
	return nil
}

// Uninstall uninstalls a TAR package
func (i *elasticAgentTARPackage) Uninstall() error {
	args := []string{"/elastic-agent/elastic-agent", "uninstall", "-f"}
	_, err := i.Exec(args)
	if err != nil {
		return fmt.Errorf("Failed to uninstall the agent with subcommand: %v", err)
	}
	return nil
}
