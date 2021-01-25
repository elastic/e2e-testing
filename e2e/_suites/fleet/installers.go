package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// InstallerPackage represents the operations that can be performed by an installer package type
type InstallerPackage interface {
	Install(containerName string, token string) error
	InstallCerts() error
	Postinstall() error
	Preinstall() error
	Uninstall() error
}

// BasePackage holds references to basic state for all installers
type BasePackage struct {
	binaryName string
	image      string
	profile    string
	service    string
}

// Postinstall executes operations after installing a DEB package
func (i *BasePackage) Postinstall() error {
	err := systemctlRun(i.profile, i.image, i.service, "enable")
	if err != nil {
		return err
	}
	return systemctlRun(i.profile, i.image, i.service, "start")
}

// DEBPackage implements operations for a DEB installer
type DEBPackage struct {
	BasePackage
}

// NewDEBPackage creates an instance for the DEB installer
func NewDEBPackage(binaryName string, profile string, image string, service string) *DEBPackage {
	return &DEBPackage{
		BasePackage: BasePackage{
			binaryName: binaryName,
			image:      image,
			profile:    profile,
			service:    service,
		},
	}
}

// Install installs a DEB package
func (i *DEBPackage) Install(containerName string, token string) error {
	cmds := []string{"apt", "install", "/" + i.binaryName, "-y"}
	return extractPackage(i.profile, i.image, i.service, cmds)
}

// InstallCerts installs the certificates for a DEB package
func (i *DEBPackage) InstallCerts() error {
	if err := execCommandInService(i.profile, i.image, i.service, []string{"apt-get", "update"}, false); err != nil {
		return err
	}
	if err := execCommandInService(i.profile, i.image, i.service, []string{"apt", "install", "ca-certificates", "-y"}, false); err != nil {
		return err
	}
	if err := execCommandInService(i.profile, i.image, i.service, []string{"update-ca-certificates"}, false); err != nil {
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

// RPMPackage implements operations for a RPM installer
type RPMPackage struct {
	BasePackage
}

// NewRPMPackage creates an instance for the RPM installer
func NewRPMPackage(binaryName string, profile string, image string, service string) *RPMPackage {
	return &RPMPackage{
		BasePackage: BasePackage{
			binaryName: binaryName,
			image:      image,
			profile:    profile,
			service:    service,
		},
	}
}

// Install installs a RPM package
func (i *RPMPackage) Install(containerName string, token string) error {
	cmds := []string{"yum", "localinstall", "/" + i.binaryName, "-y"}
	return extractPackage(i.profile, i.image, i.service, cmds)
}

// InstallCerts installs the certificates for a RPM package
func (i *RPMPackage) InstallCerts() error {
	if err := execCommandInService(i.profile, i.image, i.service, []string{"yum", "check-update"}, false); err != nil {
		return err
	}
	if err := execCommandInService(i.profile, i.image, i.service, []string{"yum", "install", "ca-certificates", "-y"}, false); err != nil {
		return err
	}
	if err := execCommandInService(i.profile, i.image, i.service, []string{"update-ca-trust", "force-enable"}, false); err != nil {
		return err
	}
	if err := execCommandInService(i.profile, i.image, i.service, []string{"update-ca-trust", "extract"}, false); err != nil {
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
	arch     string
	artifact string
	OS       string
	version  string
}

// NewTARPackage creates an instance for the RPM installer
func NewTARPackage(binaryName string, profile string, image string, service string) *TARPackage {
	return &TARPackage{
		BasePackage: BasePackage{
			binaryName: binaryName,
			image:      image,
			profile:    profile,
			service:    service,
		},
	}
}

// Install installs a TAR package
func (i *TARPackage) Install(containerName string, token string) error {
	// install the elastic-agent to /usr/bin/elastic-agent using command
	binary := fmt.Sprintf("/elastic-agent/%s", i.artifact)
	args := []string{"--force", "--insecure", "--enrollment-token", token, "--kibana-url", "http://kibana:5601"}

	err := runElasticAgentCommand(i.profile, i.image, i.service, binary, "install", args)
	if err != nil {
		return fmt.Errorf("Failed to install the agent with subcommand: %v", err)
	}

	return nil
}

// InstallCerts installs the certificates for a TAR package
func (i *TARPackage) InstallCerts() error {
	if err := execCommandInService(i.profile, i.image, i.service, []string{"apt-get", "update"}, false); err != nil {
		return err
	}
	if err := execCommandInService(i.profile, i.image, i.service, []string{"apt", "install", "ca-certificates", "-y"}, false); err != nil {
		return err
	}
	if err := execCommandInService(i.profile, i.image, i.service, []string{"update-ca-certificates"}, false); err != nil {
		return err
	}

	return nil
}

// Postinstall executes operations after installing a TAR package
func (i *TARPackage) Postinstall() error {
	log.Trace("No postinstall commands for TAR installer")
	return nil
}

// Preinstall executes operations before installing a TAR package
func (i *TARPackage) Preinstall() error {
	err := extractPackage(i.profile, i.image, i.service, []string{"tar", "-xvf", "/" + i.binaryName})
	if err != nil {
		return err
	}

	version := checkElasticAgentVersion(i.version)

	// simplify layout
	cmds := []string{"mv", fmt.Sprintf("/%s-%s-%s-%s", i.artifact, version, i.OS, i.arch), "/elastic-agent"}
	err = execCommandInService(i.profile, i.image, i.service, cmds, false)
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

// WithVersion sets the version
func (i *TARPackage) WithVersion(version string) *TARPackage {
	i.version = version
	return i
}
