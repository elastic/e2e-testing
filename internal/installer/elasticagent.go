// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/utils"
)

<<<<<<< HEAD
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

=======
>>>>>>> f043003 (Feat installer rework 2 (#1208))
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
