// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/e2e-testing/cli/docker"
	"github.com/elastic/e2e-testing/e2e"
	"github.com/elastic/e2e-testing/e2e/steps"
	log "github.com/sirupsen/logrus"
)

// InstallerPackage represents the operations that can be performed by an installer package type
type InstallerPackage interface {
	Install(cfg *FleetConfig) error
	InstallCerts() error
	PrintLogs(containerName string) error
	Postinstall() error
	Preinstall() error
	Uninstall() error
}

// BasePackage holds references to basic state for all installers
type BasePackage struct {
	binaryName string
	commitFile string
	image      string
	logFile    string
	profile    string
	service    string
}

// extractPackage depends on the underlying OS, so 'cmds' must contain the specific instructions for the OS
func (i *BasePackage) extractPackage(cmds []string) error {
	err := steps.ExecCommandInService(i.profile, i.image, i.service, cmds, profileEnv, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmds,
			"error":   err,
			"image":   i.image,
			"service": i.service,
		}).Error("Could not extract agent package in the box")

		return err
	}

	return nil
}

// Postinstall executes operations after installing a DEB package
func (i *BasePackage) Postinstall() error {
	err := systemctlRun(i.profile, i.image, i.service, "enable")
	if err != nil {
		return err
	}
	return systemctlRun(i.profile, i.image, i.service, "start")
}

// PrintLogs prints logs for the agent
func (i *BasePackage) PrintLogs(containerName string) error {
	err := i.resolveLogFile(containerName)
	if err != nil {
		return fmt.Errorf("Could not resolve log file: %v", err)
	}

	cmd := []string{
		"cat", i.logFile,
	}

	err = steps.ExecCommandInService(i.profile, i.image, i.service, cmd, profileEnv, false)
	if err != nil {
		return err
	}

	return nil
}

// resolveLogFile retrieves the full path of the log file in the underlying Docker container
// calculating the hash commit if necessary
func (i *BasePackage) resolveLogFile(containerName string) error {
	if strings.Contains(i.logFile, "%s") {
		hash, err := getElasticAgentHash(containerName, i.commitFile)
		if err != nil {
			log.WithFields(log.Fields{
				"containerName": containerName,
				"error":         err,
			}).Error("Could not get agent hash in the container")

			return err
		}

		i.logFile = fmt.Sprintf(i.logFile, hash)
	}

	return nil
}

// DEBPackage implements operations for a DEB installer
type DEBPackage struct {
	BasePackage
}

// NewDEBPackage creates an instance for the DEB installer
func NewDEBPackage(binaryName string, profile string, image string, service string, commitFile string, logFile string) *DEBPackage {
	return &DEBPackage{
		BasePackage: BasePackage{
			binaryName: binaryName,
			commitFile: commitFile,
			image:      image,
			profile:    profile,
			service:    service,
		},
	}
}

// Install installs a DEB package
func (i *DEBPackage) Install(cfg *FleetConfig) error {
	return i.extractPackage([]string{"apt", "install", "/" + i.binaryName, "-y"})
}

// InstallCerts installs the certificates for a DEB package
func (i *DEBPackage) InstallCerts() error {
	return installCertsForDebian(i.profile, i.image, i.service)
}
func installCertsForDebian(profile string, image string, service string) error {
	if err := steps.ExecCommandInService(profile, image, service, []string{"apt-get", "update"}, profileEnv, false); err != nil {
		return err
	}
	if err := steps.ExecCommandInService(profile, image, service, []string{"apt", "install", "ca-certificates", "-y"}, profileEnv, false); err != nil {
		return err
	}
	if err := steps.ExecCommandInService(profile, image, service, []string{"update-ca-certificates", "-f"}, profileEnv, false); err != nil {
		return err
	}
	return nil
}

// Preinstall executes operations before installing a DEB package
func (i *DEBPackage) Preinstall() error {
	log.Trace("No preinstall commands for DEB packages")
	return nil
}

// Uninstall uninstalls a DEB package
func (i *DEBPackage) Uninstall() error {
	log.Trace("No uninstall commands for DEB packages")
	return nil
}

// DockerPackage implements operations for a DEB installer
type DockerPackage struct {
	BasePackage
	installerPath string
	ubi8          bool
	// optional fields
	arch            string
	artifact        string
	originalVersion string
	OS              string
	version         string
}

// NewDockerPackage creates an instance for the Docker installer
func NewDockerPackage(binaryName string, profile string, image string, service string, installerPath string, ubi8 bool, commitFile string, logFile string) *DockerPackage {
	return &DockerPackage{
		BasePackage: BasePackage{
			binaryName: binaryName,
			commitFile: commitFile,
			image:      image,
			logFile:    logFile,
			profile:    profile,
			service:    service,
		},
		installerPath: installerPath,
		ubi8:          ubi8,
	}
}

// Install installs a Docker package
func (i *DockerPackage) Install(cfg *FleetConfig) error {
	log.Trace("No install commands for Docker packages")
	return nil
}

// InstallCerts installs the certificates for a Docker package
func (i *DockerPackage) InstallCerts() error {
	log.Trace("No install certs commands for Docker packages")
	return nil
}

// Preinstall executes operations before installing a Docker package
func (i *DockerPackage) Preinstall() error {
	err := docker.LoadImage(i.installerPath)
	if err != nil {
		return err
	}

	// we need to tag the loaded image because its tag relates to the target branch
	return docker.TagImage(
		"docker.elastic.co/beats/"+i.artifact+":"+agentVersionBase,
		"docker.elastic.co/observability-ci/"+i.artifact+":"+i.originalVersion+"-amd64",
	)
}

// Postinstall executes operations after installing a Docker package
func (i *DockerPackage) Postinstall() error {
	log.Trace("No postinstall commands for Docker packages")
	return nil
}

// Uninstall uninstalls a Docker package
func (i *DockerPackage) Uninstall() error {
	log.Trace("No uninstall commands for Docker packages")
	return nil
}

// WithArch sets the architecture
func (i *DockerPackage) WithArch(arch string) *DockerPackage {
	i.arch = arch
	return i
}

// WithArtifact sets the artifact
func (i *DockerPackage) WithArtifact(artifact string) *DockerPackage {
	i.artifact = artifact
	return i
}

// WithOS sets the OS
func (i *DockerPackage) WithOS(OS string) *DockerPackage {
	i.OS = OS
	return i
}

// WithVersion sets the version
func (i *DockerPackage) WithVersion(version string) *DockerPackage {
	i.version = e2e.CheckPRVersion(version, agentVersionBase) // sanitize version
	i.originalVersion = version
	return i
}

// RPMPackage implements operations for a RPM installer
type RPMPackage struct {
	BasePackage
}

// NewRPMPackage creates an instance for the RPM installer
func NewRPMPackage(binaryName string, profile string, image string, service string, commitFile string, logFile string) *RPMPackage {
	return &RPMPackage{
		BasePackage: BasePackage{
			binaryName: binaryName,
			commitFile: commitFile,
			image:      image,
			logFile:    logFile,
			profile:    profile,
			service:    service,
		},
	}
}

// Install installs a RPM package
func (i *RPMPackage) Install(cfg *FleetConfig) error {
	return i.extractPackage([]string{"yum", "localinstall", "/" + i.binaryName, "-y"})
}

// InstallCerts installs the certificates for a RPM package
func (i *RPMPackage) InstallCerts() error {
	return installCertsForCentos(i.profile, i.image, i.service)
}
func installCertsForCentos(profile string, image string, service string) error {
	if err := steps.ExecCommandInService(profile, image, service, []string{"yum", "check-update"}, profileEnv, false); err != nil {
		return err
	}
	if err := steps.ExecCommandInService(profile, image, service, []string{"yum", "install", "ca-certificates", "-y"}, profileEnv, false); err != nil {
		return err
	}
	if err := steps.ExecCommandInService(profile, image, service, []string{"update-ca-trust", "force-enable"}, profileEnv, false); err != nil {
		return err
	}
	if err := steps.ExecCommandInService(profile, image, service, []string{"update-ca-trust", "extract"}, profileEnv, false); err != nil {
		return err
	}
	return nil
}

// Preinstall executes operations before installing a RPM package
func (i *RPMPackage) Preinstall() error {
	log.Trace("No preinstall commands for RPM packages")
	return nil
}

// Uninstall uninstalls a RPM package
func (i *RPMPackage) Uninstall() error {
	log.Trace("No uninstall commands for RPM packages")
	return nil
}

// TARPackage implements operations for a RPM installer
type TARPackage struct {
	BasePackage
	// optional fields
	arch      string
	artifact  string
	OS        string
	OSFlavour string // at this moment, centos or debian
	version   string
}

// NewTARPackage creates an instance for the RPM installer
func NewTARPackage(binaryName string, profile string, image string, service string, commitFile string, logFile string) *TARPackage {
	return &TARPackage{
		BasePackage: BasePackage{
			binaryName: binaryName,
			commitFile: commitFile,
			image:      image,
			logFile:    logFile,
			profile:    profile,
			service:    service,
		},
	}
}

// Install installs a TAR package
func (i *TARPackage) Install(cfg *FleetConfig) error {
	// install the elastic-agent to /usr/bin/elastic-agent using command
	binary := fmt.Sprintf("/elastic-agent/%s", i.artifact)

	if cfg.isFleetServer() {
		return bootstrapFleetServer(i.profile, i.image, i.service, binary, cfg)
	}

	args := cfg.flags()

	err := runElasticAgentCommand(i.profile, i.image, i.service, binary, "install", args)
	if err != nil {
		return fmt.Errorf("Failed to install the agent with subcommand: %v", err)
	}

	return nil
}

// InstallCerts installs the certificates for a TAR package, using the right OS package manager
func (i *TARPackage) InstallCerts() error {
	if i.OSFlavour == "centos" {
		return installCertsForCentos(i.profile, i.image, i.service)
	} else if i.OSFlavour == "debian" {
		return installCertsForDebian(i.profile, i.image, i.service)
	}

	log.WithFields(log.Fields{
		"arch":      i.arch,
		"OS":        i.OS,
		"OSFlavour": i.OSFlavour,
	}).Debug("Installation of certificates was skipped because of unknown OS flavour")

	return nil
}

// Postinstall executes operations after installing a TAR package
func (i *TARPackage) Postinstall() error {
	log.Trace("No postinstall commands for TAR installer")
	return nil
}

// Preinstall executes operations before installing a TAR package
func (i *TARPackage) Preinstall() error {
	err := i.extractPackage([]string{"tar", "-xvf", "/" + i.binaryName})
	if err != nil {
		return err
	}

	// simplify layout
	cmds := [][]string{
		{"rm", "-fr", "/elastic-agent"},
		{"mv", fmt.Sprintf("/%s-%s-%s-%s", i.artifact, i.version, i.OS, i.arch), "/elastic-agent"},
	}
	for _, cmd := range cmds {
		err = steps.ExecCommandInService(i.profile, i.image, i.service, cmd, profileEnv, false)
		if err != nil {
			log.WithFields(log.Fields{
				"command": cmd,
				"error":   err,
				"image":   i.image,
				"service": i.service,
				"version": i.version,
			}).Error("Could not extract agent package in the box")

			return err
		}
	}

	return nil
}

// Uninstall uninstalls a TAR package
func (i *TARPackage) Uninstall() error {
	args := []string{"-f"}

	return runElasticAgentCommand(i.profile, i.image, i.service, ElasticAgentProcessName, "uninstall", args)
}

// WithArch sets the architecture
func (i *TARPackage) WithArch(arch string) *TARPackage {
	i.arch = arch
	return i
}

// WithArtifact sets the artifact
func (i *TARPackage) WithArtifact(artifact string) *TARPackage {
	i.artifact = artifact
	return i
}

// WithOS sets the OS
func (i *TARPackage) WithOS(OS string) *TARPackage {
	i.OS = OS
	return i
}

// WithOSFlavour sets the OS flavour, at this moment centos or debian
func (i *TARPackage) WithOSFlavour(OSFlavour string) *TARPackage {
	i.OSFlavour = OSFlavour
	return i
}

// WithVersion sets the version
func (i *TARPackage) WithVersion(version string) *TARPackage {
	i.version = version
	return i
}

// getElasticAgentHash uses Elastic Agent's home dir to read the file with agent's build hash
// it will return the first six characters of the hash (short hash)
func getElasticAgentHash(containerName string, commitFile string) (string, error) {
	cmd := []string{
		"cat", commitFile,
	}

	fullHash, err := docker.ExecCommandIntoContainer(context.Background(), containerName, "root", cmd)
	if err != nil {
		return "", err
	}

	runes := []rune(fullHash)
	shortHash := string(runes[0:6])

	log.WithFields(log.Fields{
		"commitFile":    commitFile,
		"containerName": containerName,
		"hash":          fullHash,
		"shortHash":     shortHash,
	}).Debug("Agent build hash found")

	return shortHash, nil
}
