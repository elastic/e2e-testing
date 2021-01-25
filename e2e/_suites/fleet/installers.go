package main

import log "github.com/sirupsen/logrus"

// InstallerPackage represents the operations that can be performed by an installer package type
type InstallerPackage interface {
	Install(containerName string, token string) error
	InstallCerts() error
	Postinstall() error
	Preinstall() error
	Uninstall() error
}

// RPMPackage implements operations for a RPM installer
type RPMPackage struct {
	binaryName string
	image      string
	profile    string
	service    string
}

// NewRPMPackage creates an instance for the RPM installer
func NewRPMPackage(binaryName string, profile string, image string, service string) *RPMPackage {
	return &RPMPackage{
		binaryName: binaryName,
		image:      image,
		profile:    profile,
		service:    service,
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

// Postinstall executes operations after installing a RPM package
func (i *RPMPackage) Postinstall() error {
	err := systemctlRun(i.profile, i.image, i.service, "enable")
	if err != nil {
		return err
	}
	return systemctlRun(i.profile, i.image, i.service, "start")
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
