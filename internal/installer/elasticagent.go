// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"context"
	"fmt"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

// ElasticAgentInstaller represents how to install an agent, depending of the box type
type ElasticAgentInstaller struct {
	artifactArch      string // architecture of the artifact
	artifactExtension string // extension of the artifact
	artifactName      string // name of the artifact
	artifactOS        string // OS of the artifact
	artifactVersion   string // version of the artifact
	BinaryPath        string // the local path where the agent for the binary is located
	EnrollFn          func(cfg *kibana.FleetConfig) error
	Image             string // docker image
	InstallerType     string
	InstallFn         func(cfg *kibana.FleetConfig) error
	InstallCertsFn    func() error
	Name              string // the name for the binary
	processName       string // name of the elastic-agent process
	Profile           string // parent docker-compose file
	PostInstallFn     func() error
	PreInstallFn      func() error
	PrintLogsFn       func(containerName string) error
	Service           string // name of the service
	Tag               string // docker tag
	UninstallFn       func() error
	workingDir        string // location of the application
}

// ListElasticAgentWorkingDirContent list Elastic Agent's working dir content
func (i *ElasticAgentInstaller) ListElasticAgentWorkingDirContent(containerName string) (string, error) {
	cmd := []string{
		"ls", "-l", i.workingDir,
	}

	content, err := deploy.ExecCommandIntoContainer(context.Background(), deploy.NewServiceRequest(containerName), "root", cmd)
	if err != nil {
		return "", err
	}

	log.WithFields(log.Fields{
		"workingDir":    i.workingDir,
		"containerName": containerName,
		"content":       content,
	}).Debug("Agent working dir content")

	return content, nil
}

// runElasticAgentCommandEnv runs a command for the elastic-agent
func runElasticAgentCommandEnv(profile deploy.ServiceRequest, image deploy.ServiceRequest, service string, process string, command string, arguments []string, env map[string]string) error {
	cmds := []string{
		"timeout", fmt.Sprintf("%dm", utils.TimeoutFactor), process, command,
	}
	cmds = append(cmds, arguments...)

	for k, v := range env {
		common.ProfileEnv[k] = v
	}

	sm := deploy.NewServiceManager()
	err := sm.ExecCommandInService(profile, image, service, cmds, common.ProfileEnv, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmds,
			"profile": profile,
			"service": service,
			"error":   err,
		}).Error("Could not run agent command in the box")

		return err
	}

	return nil
}

// downloadAgentBinary it downloads the binary and stores the location of the downloaded file
// into the installer struct, to be used else where
// If the environment variable ELASTIC_AGENT_DOWNLOAD_URL exists, then the artifact to be downloaded will
// be defined by that value
// Else if the environment variable BEATS_LOCAL_PATH is set, then the artifact
// to be used will be defined by the local snapshot produced by the local build.
// Else, if the environment variable BEATS_USE_CI_SNAPSHOTS is set, then the artifact
// to be downloaded will be defined by the latest snapshot produced by the Beats CI.
func downloadAgentBinary(artifactName string, artifact string, version string) (string, error) {
	imagePath, err := utils.FetchBeatsBinary(artifactName, artifact, version, common.BeatVersionBase, utils.TimeoutFactor, true)
	if err != nil {
		return "", err
	}

	return imagePath, nil
}

// GetElasticAgentInstaller returns an installer from a docker image
func GetElasticAgentInstaller(image string, installerType string, version string, index int) ElasticAgentInstaller {
	log.WithFields(log.Fields{
		"image":     image,
		"installer": installerType,
		"version":   version,
	}).Debug("Configuring installer for the agent")

	var installer ElasticAgentInstaller
	var err error
	if "centos" == image && "tar" == installerType {
		installer, err = newTarInstaller("centos", "latest", version, index)
	} else if "centos" == image && "rpm" == installerType {
		installer, err = newCentosInstaller("centos", "latest", version)
	} else if "debian" == image && "tar" == installerType {
		installer, err = newTarInstaller("debian", "stretch", version, index)
	} else if "debian" == image && "deb" == installerType {
		installer, err = newDebianInstaller("debian", "stretch", version)
	} else if "docker" == image && "default" == installerType {
		installer, err = newDockerInstaller(false, version)
	} else if "docker" == image && "ubi8" == installerType {
		installer, err = newDockerInstaller(true, version)
	} else {
		log.WithFields(log.Fields{
			"image":     image,
			"installer": installerType,
		}).Fatal("Sorry, we currently do not support this installer")
		return ElasticAgentInstaller{}
	}

	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"image":     image,
			"installer": installerType,
		}).Fatal("Sorry, we could not download the installer")
	}
	return installer
}
