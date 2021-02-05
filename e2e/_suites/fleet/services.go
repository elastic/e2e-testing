// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/e2e-testing/cli/docker"
	"github.com/elastic/e2e-testing/e2e"
	log "github.com/sirupsen/logrus"
)

// ElasticAgentInstaller represents how to install an agent, depending of the box type
type ElasticAgentInstaller struct {
	artifactArch      string // architecture of the artifact
	artifactExtension string // extension of the artifact
	artifactName      string // name of the artifact
	artifactOS        string // OS of the artifact
	artifactVersion   string // version of the artifact
	binDir            string // location of the binary
	commitFile        string // elastic agent commit file
	EnrollFn          func(token string) error
	homeDir           string // elastic agent home dir
	image             string // docker image
	installerType     string
	InstallFn         func(containerName string, token string) error
	InstallCertsFn    func() error
	logFile           string // the name of the log file
	logsDir           string // location of the logs
	name              string // the name for the binary
	path              string // the local path where the agent for the binary is located
	processName       string // name of the elastic-agent process
	profile           string // parent docker-compose file
	PostInstallFn     func() error
	PreInstallFn      func() error
	service           string // name of the service
	tag               string // docker tag
	UninstallFn       func() error
	workingDir        string // location of the application
}

// listElasticAgentWorkingDirContent list Elastic Agent's working dir content
func (i *ElasticAgentInstaller) listElasticAgentWorkingDirContent(containerName string) (string, error) {
	cmd := []string{
		"ls", "-l", i.workingDir,
	}

	content, err := docker.ExecCommandIntoContainer(context.Background(), containerName, "root", cmd)
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

// getElasticAgentHash uses Elastic Agent's home dir to read the file with agent's build hash
// it will return the first six characters of the hash (short hash)
func (i *ElasticAgentInstaller) getElasticAgentHash(containerName string) (string, error) {
	commitFile := i.homeDir + i.commitFile

	return getElasticAgentHash(containerName, commitFile)
}

func getElasticAgentHash(containerName string, commitFile string) (string, error) {
	cmd := []string{
		"cat", commitFile,
	}

	fullHash, err := docker.ExecCommandIntoContainer(context.Background(), containerName, "root", cmd)
	if err != nil {
		return "", err
	}

	runes := []rune(fullHash)
	shortHash := string(runes[0:6])

	log.WithFields(log.Fields{
		"commitFile":    commitFile,
		"containerName": containerName,
		"hash":          fullHash,
		"shortHash":     shortHash,
	}).Debug("Agent build hash found")

	return shortHash, nil
}

// getElasticAgentLogs uses elastic-agent log dir to read the entire log file
func (i *ElasticAgentInstaller) getElasticAgentLogs(hostname string) error {
	containerName := hostname // name of the container, which matches the hostname

	hash, err := i.getElasticAgentHash(containerName)
	if err != nil {
		log.WithFields(log.Fields{
			"containerName": containerName,
			"error":         err,
		}).Error("Could not get agent hash in the container")

		return err
	}

	logFile := i.logsDir + i.logFile
	if strings.Contains(logFile, "%s") {
		logFile = fmt.Sprintf(logFile, hash)
	}
	cmd := []string{
		"cat", logFile,
	}

	err = execCommandInService(i.profile, i.image, i.service, cmd, false)
	if err != nil {
		log.WithFields(log.Fields{
			"containerName": containerName,
			"command":       cmd,
			"error":         err,
			"hash":          hash,
		}).Error("Could not get agent logs in the container")

		return err
	}

	return nil
}

// runElasticAgentCommand runs a command for the elastic-agent
func runElasticAgentCommand(profile string, image string, service string, process string, command string, arguments []string) error {
	cmds := []string{
		process, command,
	}
	cmds = append(cmds, arguments...)

	err := execCommandInService(profile, image, service, cmds, false)
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
	imagePath, err := e2e.FetchBeatsBinary(artifactName, artifact, version, agentVersionBase, timeoutFactor, true)
	if err != nil {
		return "", err
	}

	return imagePath, nil
}

// GetElasticAgentInstaller returns an installer from a docker image
func GetElasticAgentInstaller(image string, installerType string, version string) ElasticAgentInstaller {
	log.WithFields(log.Fields{
		"image":     image,
		"installer": installerType,
	}).Debug("Configuring installer for the agent")

	var installer ElasticAgentInstaller
	var err error
	if "centos" == image && "tar" == installerType {
		installer, err = newTarInstaller("centos", "latest", version)
	} else if "centos" == image && "systemd" == installerType {
		installer, err = newCentosInstaller("centos", "latest", version)
	} else if "debian" == image && "tar" == installerType {
		installer, err = newTarInstaller("debian", "stretch", version)
	} else if "debian" == image && "systemd" == installerType {
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

func isSystemdBased(image string) bool {
	return strings.HasSuffix(image, "-systemd")
}

// newCentosInstaller returns an instance of the Centos installer for a specific version
func newCentosInstaller(image string, tag string, version string) (ElasticAgentInstaller, error) {
	image = image + "-systemd" // we want to consume systemd boxes
	service := image
	profile := FleetProfileName

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	os := "linux"
	arch := "x86_64"
	extension := "rpm"

	binaryName := e2e.BuildArtifactName(artifact, version, agentVersionBase, os, arch, extension, false)
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

	enrollFn := func(token string) error {
		args := []string{"http://kibana:5601", token, "-f", "--insecure"}

		return runElasticAgentCommand(profile, image, service, ElasticAgentProcessName, "enroll", args)
	}

	binDir := "/var/lib/elastic-agent/data/elastic-agent-%s/"

	installerPackage := NewRPMPackage(binaryName, profile, image, service)

	return ElasticAgentInstaller{
		artifactArch:      arch,
		artifactExtension: extension,
		artifactName:      artifact,
		artifactOS:        os,
		artifactVersion:   version,
		binDir:            binDir,
		commitFile:        ".elastic-agent.active.commit",
		EnrollFn:          enrollFn,
		homeDir:           "/etc/elastic-agent/",
		image:             image,
		InstallFn:         installerPackage.Install,
		InstallCertsFn:    installerPackage.InstallCerts,
		installerType:     "rpm",
		logFile:           "elastic-agent-json.log",
		logsDir:           binDir + "logs/",
		name:              binaryName,
		path:              binaryPath,
		PostInstallFn:     installerPackage.Postinstall,
		PreInstallFn:      installerPackage.Preinstall,
		processName:       ElasticAgentProcessName,
		profile:           profile,
		service:           service,
		tag:               tag,
		UninstallFn:       installerPackage.Uninstall,
		workingDir:        "/var/lib/elastic-agent",
	}, nil
}

// newDebianInstaller returns an instance of the Debian installer for a specific version
func newDebianInstaller(image string, tag string, version string) (ElasticAgentInstaller, error) {
	image = image + "-systemd" // we want to consume systemd boxes
	service := image
	profile := FleetProfileName

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	os := "linux"
	arch := "amd64"
	extension := "deb"

	binaryName := e2e.BuildArtifactName(artifact, version, agentVersionBase, os, arch, extension, false)
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

	enrollFn := func(token string) error {
		args := []string{"http://kibana:5601", token, "-f", "--insecure"}

		return runElasticAgentCommand(profile, image, service, ElasticAgentProcessName, "enroll", args)
	}

	binDir := "/var/lib/elastic-agent/data/elastic-agent-%s/"

	installerPackage := NewDEBPackage(binaryName, profile, image, service)

	return ElasticAgentInstaller{
		artifactArch:      arch,
		artifactExtension: extension,
		artifactName:      artifact,
		artifactOS:        os,
		artifactVersion:   version,
		binDir:            binDir,
		commitFile:        ".elastic-agent.active.commit",
		EnrollFn:          enrollFn,
		homeDir:           "/etc/elastic-agent/",
		image:             image,
		InstallFn:         installerPackage.Install,
		InstallCertsFn:    installerPackage.InstallCerts,
		installerType:     "deb",
		logFile:           "elastic-agent-json.log",
		logsDir:           binDir + "logs/",
		name:              binaryName,
		path:              binaryPath,
		PostInstallFn:     installerPackage.Postinstall,
		PreInstallFn:      installerPackage.Preinstall,
		processName:       ElasticAgentProcessName,
		profile:           profile,
		service:           service,
		tag:               tag,
		UninstallFn:       installerPackage.Uninstall,
		workingDir:        "/var/lib/elastic-agent",
	}, nil
}

// newDockerInstaller returns an instance of the Docker installer
func newDockerInstaller(ubi8 bool, version string) (ElasticAgentInstaller, error) {
	image := "elastic-agent"
	service := image
	profile := FleetProfileName

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"

	artifactName := artifact
	if ubi8 {
		artifactName = "elastic-agent-ubi8"
	}

	os := "linux"
	arch := "amd64"
	extension := "tar.gz"

	binaryName := e2e.BuildArtifactName(artifactName, version, agentVersionBase, os, arch, extension, true)
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

	commitFile := ".elastic-agent.active.commit"
	homeDir := "/usr/share/elastic-agent"
	binDir := "/usr/share/elastic-agent/data/elastic-agent-%s/"

	enrollFn := func(token string) error {
		return nil
	}

	installerPackage := NewDockerPackage(binaryName, profile, image, service, binaryPath, ubi8).
		WithArch(arch).
		WithArtifact(artifact).
		WithOS(os).
		WithVersion(version)

	return ElasticAgentInstaller{
		artifactArch:      arch,
		artifactExtension: extension,
		artifactName:      artifact,
		artifactOS:        os,
		artifactVersion:   version,
		binDir:            binDir,
		commitFile:        commitFile,
		EnrollFn:          enrollFn,
		homeDir:           homeDir,
		image:             image,
		InstallFn:         installerPackage.Install,
		InstallCertsFn:    installerPackage.InstallCerts,
		installerType:     "docker",
		logFile:           "elastic-agent-json.log",
		logsDir:           binDir + "logs/",
		name:              binaryName,
		path:              binaryPath,
		PostInstallFn:     installerPackage.Postinstall,
		PreInstallFn:      installerPackage.Preinstall,
		processName:       ElasticAgentProcessName,
		profile:           profile,
		service:           service,
		tag:               version,
		UninstallFn:       installerPackage.Uninstall,
		workingDir:        "/usr/share/elastic-agent/",
	}, nil
}

// newTarInstaller returns an instance of the Debian installer for a specific version
func newTarInstaller(image string, tag string, version string) (ElasticAgentInstaller, error) {
	image = image + "-systemd" // we want to consume systemd boxes
	service := image
	profile := FleetProfileName

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	os := "linux"
	arch := "x86_64"
	extension := "tar.gz"

	binaryName := e2e.BuildArtifactName(artifact, version, agentVersionBase, os, arch, extension, false)
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

	commitFile := ".elastic-agent.active.commit"
	homeDir := "/elastic-agent/"
	binDir := "/usr/bin/"

	enrollFn := func(token string) error {
		args := []string{"http://kibana:5601", token, "-f", "--insecure"}

		return runElasticAgentCommand(profile, image, service, ElasticAgentProcessName, "enroll", args)
	}

	//
	installerPackage := NewTARPackage(binaryName, profile, image, service).
		WithArch(arch).
		WithArtifact(artifact).
		WithOS(os).
		WithVersion(version)

	return ElasticAgentInstaller{
		artifactArch:      arch,
		artifactExtension: extension,
		artifactName:      artifact,
		artifactOS:        os,
		artifactVersion:   version,
		binDir:            binDir,
		commitFile:        commitFile,
		EnrollFn:          enrollFn,
		homeDir:           homeDir,
		image:             image,
		InstallFn:         installerPackage.Install,
		InstallCertsFn:    installerPackage.InstallCerts,
		installerType:     "tar",
		logFile:           "elastic-agent.log",
		logsDir:           "/opt/Elastic/Agent/",
		name:              binaryName,
		path:              binaryPath,
		PostInstallFn:     installerPackage.Postinstall,
		PreInstallFn:      installerPackage.Preinstall,
		processName:       ElasticAgentProcessName,
		profile:           profile,
		service:           service,
		tag:               tag,
		UninstallFn:       installerPackage.Uninstall,
		workingDir:        "/opt/Elastic/Agent/",
	}, nil
}

func systemctlRun(profile string, image string, service string, command string) error {
	cmd := []string{"systemctl", command, ElasticAgentProcessName}
	err := execCommandInService(profile, image, service, cmd, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
			"service": service,
		}).Errorf("Could not %s the service", command)

		return err
	}

	log.WithFields(log.Fields{
		"command": cmd,
		"service": service,
	}).Trace("Systemctl executed")
	return nil
}
