// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

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
func (i *DEBPackage) Install(cfg *kibana.FleetConfig) error {
	return i.extractPackage([]string{"apt", "install", "/" + i.binaryName, "-y"})
}

// InstallCerts installs the certificates for a DEB package
func (i *DEBPackage) InstallCerts() error {
	return installCertsForDebian(i.profile, i.image, i.service)
}
func installCertsForDebian(profile string, image string, service string) error {
	sm := deploy.NewServiceManager()
	serviceProfile := deploy.NewServiceRequest(profile)
	serviceImage := deploy.NewServiceRequest(image)

	if err := sm.ExecCommandInService(serviceProfile, serviceImage, service, []string{"apt-get", "update"}, common.ProfileEnv, false); err != nil {
		return err
	}
	if err := sm.ExecCommandInService(serviceProfile, serviceImage, service, []string{"apt", "install", "ca-certificates", "-y"}, common.ProfileEnv, false); err != nil {
		return err
	}
	if err := sm.ExecCommandInService(serviceProfile, serviceImage, service, []string{"update-ca-certificates", "-f"}, common.ProfileEnv, false); err != nil {
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

// newDebianInstaller returns an instance of the Debian installer for a specific version
func newDebianInstaller(image string, tag string, version string) (ElasticAgentInstaller, error) {
	image = image + "-systemd" // we want to consume systemd boxes
	service := image
	profile := common.FleetProfileName

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	os := "linux"
	arch := "amd64"
	extension := "deb"

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

	profileService := deploy.NewServiceRequest(profile)
	imageService := deploy.NewServiceRequest(image).WithFlavour("debian")

	enrollFn := func(cfg *kibana.FleetConfig) error {
		return runElasticAgentCommandEnv(profileService, imageService, service, common.ElasticAgentProcessName, "enroll", cfg.Flags(), map[string]string{})
	}

	workingDir := "/var/lib/elastic-agent"
	binDir := workingDir + "/data/elastic-agent-%s/"

	commitFile := "/etc/elastic-agent/.elastic-agent.active.commit"

	logsDir := binDir + "logs/"
	logFileName := "elastic-agent-json.log"
	logFile := logsDir + "/" + logFileName

	installerPackage := NewDEBPackage(binaryName, profile, image, service, commitFile, logFile)

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
		InstallerType:     "deb",
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
