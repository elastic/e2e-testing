// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"context"
	"fmt"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/mholt/archiver/v3"
	log "github.com/sirupsen/logrus"
)

// WindowsPackage implements operations for a Windows installer
type WindowsPackage struct {
	BasePackage
	// optional fields
	arch      string
	artifact  string
	OS        string
	OSFlavour string
	version   string
}

// NewWindowsPackage creates an instance for the Windows package installer
func NewWindowsPackage(binaryName string) *WindowsPackage {
	return &WindowsPackage{
		BasePackage: BasePackage{
			binaryName: binaryName,
		},
	}
}

// Install installs a Windows package
func (i *WindowsPackage) Install(cfg *kibana.FleetConfig) error {
	args := []string{"install"}
	args = append(args, cfg.Flags()...)

	_, err := shell.Execute(context.Background(), "C:\\elastic-agent", common.GetElasticAgentProcessName(), args...)
	if err != nil {
		return fmt.Errorf("Failed to install: %v", err)
	}
	return nil
}

// InstallCerts installs the certificates for a Windows package, using the right OS package manager
func (i *WindowsPackage) InstallCerts() error {
	log.Trace("No installcerts commands for Windows installer")
	return nil
}

// Postinstall executes operations after installing a TAR package
func (i *WindowsPackage) Postinstall() error {
	log.Trace("No postinstall commands for Windows installer")
	return nil
}

// Preinstall executes operations before installing a TAR package
func (i *WindowsPackage) Preinstall() error {
	err := archiver.Unarchive(i.binaryName, "elastic-agent")
	if err != nil {
		return err
	}

	return nil
}

// Uninstall elastic-agent
func (i *WindowsPackage) Uninstall() error {
	_, err := shell.Execute(context.Background(), "C:\\elastic-agent", common.GetElasticAgentProcessName(), "uninstall", "-f")
	if err != nil {
		return err
	}
	return nil
}

// WithArch sets the architecture
func (i *WindowsPackage) WithArch(arch string) *WindowsPackage {
	i.arch = arch
	return i
}

// WithArtifact sets the artifact
func (i *WindowsPackage) WithArtifact(artifact string) *WindowsPackage {
	i.artifact = artifact
	return i
}

// WithOS sets the OS
func (i *WindowsPackage) WithOS(OS string) *WindowsPackage {
	i.OS = OS
	return i
}

// WithOSFlavour sets the OS flavour
func (i *WindowsPackage) WithOSFlavour(OSFlavour string) *WindowsPackage {
	i.OSFlavour = OSFlavour
	return i
}

// WithVersion sets the version
func (i *WindowsPackage) WithVersion(version string) *WindowsPackage {
	i.version = version
	return i
}

// newWindowsInstaller returns an instance of the Debian installer for a specific version
func newWindowsInstaller(version string) (ElasticAgentInstaller, error) {
	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	os := "windows"
	arch := "x86_64"
	extension := "zip"

	binaryName := utils.BuildArtifactName(artifact, version, common.AgentVersionBase, os, arch, extension, false)
	binaryPath, err := downloadAgentBinary(binaryName, artifact, version)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":  artifact,
			"version":   version,
			"os":        os,
			"arch":      arch,
			"extension": extension,
			"error":     err,
		}).Error("Could not download the binary for the agent")
		return ElasticAgentInstaller{}, err
	}

	workingDir := "C:\\elastic-agent"

	// logsDir := workingDir + "\\data\\elastic-agent-%s\\log\\"
	// logFileName := "elastic-agent-json.log"
	// logFile := logsDir + "\\" + logFileName

	enrollFn := func(cfg *kibana.FleetConfig) error {
		args := []string{"enroll"}
		args = append(args, cfg.Flags()...)

		_, err := shell.Execute(context.Background(), "C:\\elastic-agent", common.GetElasticAgentProcessName(), args...)
		if err != nil {
			return fmt.Errorf("Failed to enroll: %v", err)
		}
		return nil

	}

	installerPackage := NewWindowsPackage(binaryName).
		WithArch(arch).
		WithArtifact(artifact).
		WithOS(os).
		WithOSFlavour("TBD").
		WithVersion(utils.CheckPRVersion(version, common.AgentVersionBase)) // sanitize version

	return ElasticAgentInstaller{
		artifactArch:      arch,
		artifactExtension: extension,
		artifactName:      artifact,
		artifactOS:        os,
		artifactVersion:   version,
		BinaryPath:        binaryPath,
		EnrollFn:          enrollFn,
		InstallFn:         installerPackage.Install,
		InstallCertsFn:    installerPackage.InstallCerts,
		InstallerType:     "windows",
		Name:              binaryName,
		PostInstallFn:     installerPackage.Postinstall,
		PreInstallFn:      installerPackage.Preinstall,
		PrintLogsFn:       installerPackage.PrintLogs,
		processName:       common.GetElasticAgentProcessName(),
		UninstallFn:       installerPackage.Uninstall,
		workingDir:        workingDir,
	}, nil
}
