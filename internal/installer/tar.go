// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"fmt"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/compose"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

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
func (i *TARPackage) Install(cfg *kibana.FleetConfig) error {
	// install the elastic-agent to /usr/bin/elastic-agent using command
	binary := fmt.Sprintf("/elastic-agent/%s", i.artifact)
	args := cfg.Flags()

	err := runElasticAgentCommandEnv(i.profile, i.image, i.service, binary, "install", args, map[string]string{})
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
		sm := compose.NewServiceManager()
		err = sm.ExecCommandInService(i.profile, i.image, i.service, cmd, common.ProfileEnv, false)
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

	return runElasticAgentCommandEnv(i.profile, i.image, i.service, common.ElasticAgentProcessName, "uninstall", args, map[string]string{})
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

// newTarInstaller returns an instance of the Debian installer for a specific version
func newTarInstaller(image string, tag string, version string, fleetServerHost string) (ElasticAgentInstaller, error) {
	dockerImage := image + "-systemd" // we want to consume systemd boxes
	if fleetServerHost == "" {
		dockerImage = "fleet-server-" + image
	}

	service := dockerImage
	profile := common.FleetProfileName

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	os := "linux"
	arch := "x86_64"
	if utils.GetArchitecture() == "arm64" {
		arch = "arm64"
	}
	extension := "tar.gz"

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

	workingDir := "/opt/Elastic/Agent"

	commitFile := "/elastic-agent/.elastic-agent.active.commit"

	logsDir := workingDir + "/data/elastic-agent-%s/logs/"
	logFileName := "elastic-agent-json.log"
	logFile := logsDir + "/" + logFileName

	enrollFn := func(cfg *kibana.FleetConfig) error {
		return runElasticAgentCommandEnv(profile, dockerImage, service, common.ElasticAgentProcessName, "enroll", cfg.Flags(), map[string]string{})
	}

	//
	installerPackage := NewTARPackage(binaryName, profile, dockerImage, service, commitFile, logFile).
		WithArch(arch).
		WithArtifact(artifact).
		WithOS(os).
		WithOSFlavour(image).
		WithVersion(utils.CheckPRVersion(version, common.AgentVersionBase)) // sanitize version

	return ElasticAgentInstaller{
		artifactArch:      arch,
		artifactExtension: extension,
		artifactName:      artifact,
		artifactOS:        os,
		artifactVersion:   version,
		BinaryPath:        binaryPath,
		EnrollFn:          enrollFn,
		Image:             dockerImage,
		InstallFn:         installerPackage.Install,
		InstallCertsFn:    installerPackage.InstallCerts,
		InstallerType:     "tar",
		Name:              binaryName,
		PostInstallFn:     installerPackage.Postinstall,
		PreInstallFn:      installerPackage.Preinstall,
		PrintLogsFn:       installerPackage.PrintLogs,
		processName:       common.ElasticAgentProcessName,
		Profile:           profile,
		Service:           service,
		Tag:               tag,
		UninstallFn:       installerPackage.Uninstall,
		workingDir:        workingDir,
	}, nil
}
