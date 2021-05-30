// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"fmt"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

// elasticAgentDockerPackage implements operations for a docker installer
type elasticAgentDockerPackage struct {
	service deploy.ServiceRequest
	deploy  deploy.Deployment
}

// AttachElasticAgentDockerPackage creates an instance for the docker installer
func AttachElasticAgentDockerPackage(deploy deploy.Deployment, service deploy.ServiceRequest) deploy.ServiceOperator {
	return &elasticAgentDockerPackage{
		service: service,
		deploy:  deploy,
	}
}

// AddFiles will add files into the service environment, default destination is /
func (i *elasticAgentDockerPackage) AddFiles(files []string) error {
	return i.deploy.AddFiles(i.service, files)
}

// Inspect returns info on package
func (i *elasticAgentDockerPackage) Inspect() (deploy.ServiceOperatorManifest, error) {
	return deploy.ServiceOperatorManifest{
		WorkDir:    "/usr/share/elastic-agent",
		CommitFile: "/usr/share/elastic-agent/.elastic-agent.active.commit",
	}, nil
}

// Install installs a package
func (i *elasticAgentDockerPackage) Install() error {
	return nil
}

// Exec will execute a command within the service environment
func (i *elasticAgentDockerPackage) Exec(args []string) (string, error) {
	output, err := i.deploy.ExecIn(i.service, args)
	return output, err
}

// Enroll will enroll the agent into fleet
func (i *elasticAgentDockerPackage) Enroll(token string) error {
	return nil
}

// InstallCerts installs the certificates for a package, using the right OS package manager
func (i *elasticAgentDockerPackage) InstallCerts() error {
	return nil
}

// Logs prints logs of service
func (i *elasticAgentDockerPackage) Logs() error {
	return i.deploy.Logs(i.service)
}

// Postinstall executes operations after installing a package
func (i *elasticAgentDockerPackage) Postinstall() error {
	return nil
}

// Preinstall executes operations before installing a package
func (i *elasticAgentDockerPackage) Preinstall() error {
	artifact := "elastic-agent"
	os := "linux"
	arch := "x86_64"
	extension := "tar.gz"

	binaryName := utils.BuildArtifactName(artifact, common.BeatVersion, common.BeatVersionBase, os, arch, extension, false)
	binaryPath, err := utils.FetchBeatsBinary(binaryName, artifact, common.BeatVersion, common.BeatVersionBase, utils.TimeoutFactor, true)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":  artifact,
			"version":   common.BeatVersion,
			"os":        os,
			"arch":      arch,
			"extension": extension,
			"error":     err,
		}).Error("Could not download the binary for the agent")
		return err
	}

	err = deploy.LoadImage(binaryPath)
	if err != nil {
		return err
	}

	// we need to tag the loaded image because its tag relates to the target branch
	return deploy.TagImage(
		fmt.Sprintf("docker.elastic.co/beats/%s:%s", artifact, common.BeatVersionBase),
		fmt.Sprintf("docker.elastic.co/observability-ci/%s:%s-amd64", artifact, common.BeatVersion),
	)
}

// Start will start a service
func (i *elasticAgentDockerPackage) Start() error {
	_, err := i.Exec([]string{"systemctl", "start", "elastic-agent"})
	if err != nil {
		return err
	}
	return nil
}

// Stop will start a service
func (i *elasticAgentDockerPackage) Stop() error {
	_, err := i.Exec([]string{"systemctl", "stop", "elastic-agent"})
	if err != nil {
		return err
	}
	return nil
}

// Uninstall uninstalls a TAR package
func (i *elasticAgentDockerPackage) Uninstall() error {
	args := []string{"elastic-agent", "uninstall", "-f"}
	_, err := i.Exec(args)
	if err != nil {
		return fmt.Errorf("Failed to uninstall the agent with subcommand: %v", err)
	}
	return nil
}
