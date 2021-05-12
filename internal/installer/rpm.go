// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/compose"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

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
func (i *RPMPackage) Install(cfg *kibana.FleetConfig) error {
	return i.extractPackage([]string{"yum", "localinstall", "/" + i.binaryName, "-y"})
}

// InstallCerts installs the certificates for a RPM package
func (i *RPMPackage) InstallCerts() error {
	return installCertsForCentos(i.profile, i.image, i.service)
}
func installCertsForCentos(profile string, image string, service string) error {
	sm := compose.NewServiceManager()
	if err := sm.ExecCommandInService(profile, image, service, []string{"yum", "check-update"}, common.ProfileEnv, false); err != nil {
		return err
	}
	if err := sm.ExecCommandInService(profile, image, service, []string{"yum", "install", "ca-certificates", "-y"}, common.ProfileEnv, false); err != nil {
		return err
	}
	if err := sm.ExecCommandInService(profile, image, service, []string{"update-ca-trust", "force-enable"}, common.ProfileEnv, false); err != nil {
		return err
	}
	if err := sm.ExecCommandInService(profile, image, service, []string{"update-ca-trust", "extract"}, common.ProfileEnv, false); err != nil {
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

// newCentosInstaller returns an instance of the Centos installer for a specific version
func newCentosInstaller(image string, tag string, version string) (ElasticAgentInstaller, error) {
	image = image + "-systemd" // we want to consume systemd boxes
	service := image
	profile := common.FleetProfileName

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	os := "linux"
	arch := "x86_64"
	if utils.GetArchitecture() == "arm64" {
		arch = "aarch64"
	}
	extension := "rpm"

	binaryName := utils.BuildArtifactName(artifact, version, common.BeatVersionBase, os, arch, extension, false)
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

	enrollFn := func(cfg *kibana.FleetConfig) error {
		return runElasticAgentCommandEnv(profile, image, service, common.ElasticAgentProcessName, "enroll", cfg.Flags(), map[string]string{})
	}

	workingDir := "/var/lib/elastic-agent"
	binDir := workingDir + "/data/elastic-agent-%s/"

	commitFile := "/etc/elastic-agent/.elastic-agent.active.commit"

	logsDir := binDir + "logs/"
	logFileName := "elastic-agent-json.log"
	logFile := logsDir + "/" + logFileName

	installerPackage := NewRPMPackage(binaryName, profile, image, service, commitFile, logFile)

	return ElasticAgentInstaller{
		artifactArch:      arch,
		artifactExtension: extension,
		artifactName:      artifact,
		artifactOS:        os,
		artifactVersion:   version,
		BinaryPath:        binaryPath,
		EnrollFn:          enrollFn,
		Image:             image,
		InstallFn:         installerPackage.Install,
		InstallCertsFn:    installerPackage.InstallCerts,
		InstallerType:     "rpm",
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
