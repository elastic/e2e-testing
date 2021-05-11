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

// elasticAgentDEBPackage implements operations for a DEB installer
type elasticAgentDEBPackage struct {
	service  string
	executor func(service string, cmd []string) (string, error)
	copier   func(service string, files []string) error
}

// NewElasticAgentDEBPackage creates an instance for the DEB installer
func NewElasticAgentDEBPackage(service string, executor func(service string, cmd []string) (string, error), copier func(service string, files []string) error) Package {
	return &elasticAgentDEBPackage{
		service:  service,
		executor: executor,
		copier:   copier,
	}
}

// AddFiles will add files into the service environment, default destination is /
func (i *elasticAgentDEBPackage) AddFiles(files []string) error {
	return i.copier(i.service, files)
}

// Inspect returns info on package
func (i *elasticAgentDEBPackage) Inspect() (PackageManifest, error) {
	return PackageManifest{
		WorkDir:    "/var/lib/elastic-agent",
		CommitFile: "/etc/elastic-agent/.elastic-agent.active.commit",
	}, nil
}

// Install installs a DEB package
func (i *elasticAgentDEBPackage) Install() error {
	log.Trace("No additional install commands for DEB")
	return nil
}

// Exec will execute a command within the service environment
func (i *elasticAgentDEBPackage) Exec(args []string) (string, error) {
	output, err := i.executor(i.service, args)
	return output, err
}

// Enroll will enroll the agent into fleet
func (i *elasticAgentDEBPackage) Enroll(token string) error {

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

// InstallCerts installs the certificates for a DEB package, using the right OS package manager
func (i *elasticAgentDEBPackage) InstallCerts() error {
	cmds := [][]string{
		{"apt-get", "update"},
		{"apt", "install", "ca-certificates", "-y"},
		{"update-ca-certificates", "-f"},
	}
	for _, cmd := range cmds {
		if _, err := i.executor(i.service, cmd); err != nil {
			return err
		}
	}
	return nil
}

// Logs prints logs of service
func (i *elasticAgentDEBPackage) Logs() error {
	return nil
}

// Postinstall executes operations after installing a DEB package
func (i *elasticAgentDEBPackage) Postinstall() error {
	_, err := i.executor(i.service, []string{"systemctl", "restart", "elastic-agent"})
	if err != nil {
		return err
	}
	return nil
}

// Preinstall executes operations before installing a DEB package
func (i *elasticAgentDEBPackage) Preinstall() error {
	artifact := "elastic-agent"
	os := "linux"
	arch := "amd64"
	extension := "deb"

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

	_, err = i.executor(i.service, []string{"apt", "install", "/" + binaryName, "-y"})
	if err != nil {
		return err
	}

	return nil
}

// Start will start a service
func (i *elasticAgentDEBPackage) Start() error {
	_, err := i.Exec([]string{"systemctl", "start", "elastic-agent"})
	if err != nil {
		return err
	}
	return nil
}

// Stop will start a service
func (i *elasticAgentDEBPackage) Stop() error {
	_, err := i.Exec([]string{"systemctl", "stop", "elastic-agent"})
	if err != nil {
		return err
	}
	return nil
}

// Uninstall uninstalls a DEB package
func (i *elasticAgentDEBPackage) Uninstall() error {
	args := []string{"elastic-agent", "uninstall", "-f"}
	_, err := i.Exec(args)
	if err != nil {
		return fmt.Errorf("Failed to uninstall the agent with subcommand: %v", err)
	}
	return nil
}
