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

// elasticAgentRPMPackage implements operations for a RPM installer
type elasticAgentRPMPackage struct {
	service  string
	executor func(service string, cmd []string) (string, error)
	copier   func(service string, files []string) error
}

// NewElasticAgentRPMPackage creates an instance for the RPM installer
func NewElasticAgentRPMPackage(service string, executor func(service string, cmd []string) (string, error), copier func(service string, files []string) error) Package {
	return &elasticAgentRPMPackage{
		service:  service,
		executor: executor,
		copier:   copier,
	}
}

// AddFiles will add files into the service environment, default destination is /
func (i *elasticAgentRPMPackage) AddFiles(files []string) error {
	return i.copier(i.service, files)
}

// Inspect returns info on package
func (i *elasticAgentRPMPackage) Inspect() (PackageManifest, error) {
	return PackageManifest{
		WorkDir:    "/var/lib/elastic-agent",
		CommitFile: "/etc/elastic-agent/.elastic-agent.active.commit",
	}, nil
}

// Install installs a RPM package
func (i *elasticAgentRPMPackage) Install() error {
	log.Trace("No additional install commands for RPM")
	return nil
}

// Exec will execute a command within the service environment
func (i *elasticAgentRPMPackage) Exec(args []string) (string, error) {
	output, err := i.executor(i.service, args)
	return output, err
}

// Enroll will enroll the agent into fleet
func (i *elasticAgentRPMPackage) Enroll(token string) error {

	cfg, _ := kibana.NewFleetConfig(token)
	args := []string{"elastic-agent", "enroll"}
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

// InstallCerts installs the certificates for a RPM package, using the right OS package manager
func (i *elasticAgentRPMPackage) InstallCerts() error {
	cmds := [][]string{
		{"yum", "check-update"},
		{"yum", "install", "ca-certificates", "-y"},
		{"update-ca-trust", "force-enable"},
		{"update-ca-trust", "extract"},
	}
	for _, cmd := range cmds {
		if _, err := i.executor(i.service, cmd); err != nil {
			return err
		}
	}
	return nil
}

// Logs prints logs of service
func (i *elasticAgentRPMPackage) Logs() error {
	return nil
}

// Postinstall executes operations after installing a RPM package
func (i *elasticAgentRPMPackage) Postinstall() error {
	_, err := i.executor(i.service, []string{"systemctl", "restart", "elastic-agent"})
	if err != nil {
		return err
	}
	return nil
}

// Preinstall executes operations before installing a RPM package
func (i *elasticAgentRPMPackage) Preinstall() error {
	artifact := "elastic-agent"
	os := "linux"
	arch := "x86_64"
	extension := "rpm"

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

	_, err = i.executor(i.service, []string{"yum", "localinstall", "/" + binaryName, "-y"})
	if err != nil {
		return err
	}

	return nil
}

// Start will start a service
func (i *elasticAgentRPMPackage) Start() error {
	_, err := i.Exec([]string{"systemctl", "start", "elastic-agent"})
	if err != nil {
		return err
	}
	return nil
}

// Stop will start a service
func (i *elasticAgentRPMPackage) Stop() error {
	_, err := i.Exec([]string{"systemctl", "stop", "elastic-agent"})
	if err != nil {
		return err
	}
	return nil
}

// Uninstall uninstalls a RPM package
func (i *elasticAgentRPMPackage) Uninstall() error {
	args := []string{"elastic-agent", "uninstall", "-f"}
	_, err := i.Exec(args)
	if err != nil {
		return fmt.Errorf("Failed to uninstall the agent with subcommand: %v", err)
	}
	return nil
}
