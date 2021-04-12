// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/docker"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

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
func (i *DockerPackage) Install(containerName string, token string) error {
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
		"docker.elastic.co/beats/"+i.artifact+":"+common.AgentVersionBase,
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
	i.version = utils.CheckPRVersion(version, common.AgentVersionBase) // sanitize version
	i.originalVersion = version
	return i
}

// newDockerInstaller returns an instance of the Docker installer
func newDockerInstaller(ubi8 bool, version string) (ElasticAgentInstaller, error) {
	image := "elastic-agent"
	service := image
	profile := common.FleetProfileName

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"

	artifactName := artifact
	if ubi8 {
		artifactName = "elastic-agent-ubi8"
		image = "elastic-agent-ubi8"
	}

	os := "linux"
	arch := "amd64"
	extension := "tar.gz"

	binaryName := utils.BuildArtifactName(artifactName, version, common.AgentVersionBase, os, arch, extension, true)
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

	homeDir := "/usr/share/elastic-agent"
	workingDir := homeDir
	binDir := homeDir + "/data/elastic-agent-%s/"

	commitFile := homeDir + ".elastic-agent.active.commit"

	logsDir := binDir + "logs/"
	logFileName := "elastic-agent-json.log"
	logFile := logsDir + "/" + logFileName

	enrollFn := func(token string) error {
		return nil
	}

	installerPackage := NewDockerPackage(binaryName, profile, artifactName, service, binaryPath, ubi8, commitFile, logFile).
		WithArch(arch).
		WithArtifact(artifactName).
		WithOS(os).
		WithVersion(version)

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
		InstallerType:     "docker",
		Name:              binaryName,
		PostInstallFn:     installerPackage.Postinstall,
		PreInstallFn:      installerPackage.Preinstall,
		PrintLogsFn:       installerPackage.PrintLogs,
		processName:       common.ElasticAgentProcessName,
		Profile:           profile,
		Service:           service,
		Tag:               version,
		UninstallFn:       installerPackage.Uninstall,
		workingDir:        workingDir,
	}, nil
}
