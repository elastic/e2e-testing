package main

import (
	"fmt"

	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/cli/shell"
	"github.com/elastic/e2e-testing/e2e"
	log "github.com/sirupsen/logrus"
)

// agentVersion is the version of the agent to use
// It can be overriden by ELASTIC_AGENT_VERSION env var
var agentVersion = "7.9.0"

func init() {
	config.Init()

	agentVersion = shell.GetEnv("ELASTIC_AGENT_VERSION", agentVersion)
}

// ElasticAgentInstaller represents how to install an agent, depending of the box type
type ElasticAgentInstaller struct {
	artifactArch      string // architecture of the artifact
	artifactExtension string // extension of the artifact
	artifactName      string // name of the artifact
	artifactOS        string // OS of the artifact
	artifactVersion   string // version of the artifact
	image             string // docker image
	InstallCmds       []string
	name              string // the name for the binary
	path              string // the local path where the agent for the binary is located
	profile           string // parent docker-compose file
	PostInstallFn     func() error
	service           string // name of the service
	tag               string // docker tag
}

// downloadAgentBinary it downloads the binary and stores the location of the downloaded file
// into the installer struct, to be used else where
func downloadAgentBinary(artifact string, version string, os string, arch string, extension string) (string, string, error) {
	downloadURL, err := e2e.GetElasticArtifactURL(artifact, version, os, arch, extension)
	if err != nil {
		return "", "", err
	}

	fileName := fmt.Sprintf("%s-%s-%s-%s.%s", artifact, version, os, arch, extension)
	filePath, err := e2e.DownloadFile(downloadURL)

	return fileName, filePath, err
}

// GetElasticAgentInstaller returns an installer from a docker image
func GetElasticAgentInstaller(image string) ElasticAgentInstaller {
	log.WithFields(log.Fields{
		"image": image,
	}).Debug("Configuring installer for the agent")

	if "centos" == image {
		return newCentosInstaller("centos", "7")
	} else if "centos-systemd" == image {
		return newCentosInstaller("centos-systemd", "latest")
	} else if "debian-systemd" == image {
		return newDebianInstaller()
	}

	log.WithField("image", image).Fatal("Sorry, we currently do not support this installer")
	return ElasticAgentInstaller{}
}

// newCentosInstaller returns an instance of the Centos installer
func newCentosInstaller(image string, tag string) ElasticAgentInstaller {
	service := image
	profile := "ingest-manager"

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	version := agentVersion
	os := "linux"
	arch := "x86_64"
	extension := "rpm"

	binaryName, binaryPath, err := downloadAgentBinary(artifact, version, os, arch, extension)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":  artifact,
			"version":   version,
			"os":        os,
			"arch":      arch,
			"extension": extension,
			"error":     err,
		}).Error("Could not download the binary for the agent")
	}

	fn := func() error {
		return startAgent(profile, image, service)
	}
	if image == "centos-systemd" {
		fn = func() error {
			return systemctlInstall(profile, image, service)
		}
	}

	return ElasticAgentInstaller{
		artifactArch:      arch,
		artifactExtension: extension,
		artifactName:      artifact,
		artifactOS:        os,
		artifactVersion:   version,
		image:             image,
		InstallCmds:       []string{"yum", "localinstall", "/" + binaryName, "-y"},
		name:              binaryName,
		path:              binaryPath,
		PostInstallFn:     fn,
		profile:           profile,
		service:           service,
		tag:               tag,
	}
}

// newDebianInstaller returns an instance of the Debian installer
func newDebianInstaller() ElasticAgentInstaller {
	image := "debian-systemd"
	service := image
	tag := "stretch"
	profile := "ingest-manager"

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	version := agentVersion
	os := "linux"
	arch := "amd64"
	extension := "deb"

	binaryName, binaryPath, err := downloadAgentBinary(artifact, version, os, arch, extension)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":  artifact,
			"version":   version,
			"os":        os,
			"arch":      arch,
			"extension": extension,
			"error":     err,
		}).Error("Could not download the binary for the agent")
	}

	fn := func() error {
		return systemctlInstall(profile, image, service)
	}

	return ElasticAgentInstaller{
		artifactArch:      arch,
		artifactExtension: extension,
		artifactName:      artifact,
		artifactOS:        os,
		artifactVersion:   version,
		image:             image,
		InstallCmds:       []string{"apt", "install", "/" + binaryName, "-y"},
		name:              binaryName,
		path:              binaryPath,
		PostInstallFn:     fn,
		profile:           profile,
		service:           service,
		tag:               tag,
	}
}

func startAgent(profile string, image string, service string) error {
	cmd := []string{"elastic-agent", "run"}
	err := execCommandInService(profile, image, service, cmd, true)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
			"image":   image,
			"service": service,
		}).Error("Could not run the agent")

		return err
	}

	log.WithFields(log.Fields{
		"command": cmd,
		"image":   image,
		"service": service,
	}).Debug("Agent run")

	return nil
}

func systemctlInstall(profile string, image string, service string) error {
	commands := []string{"enable", "start"}

	for _, command := range commands {
		cmd := []string{"systemctl", command, "elastic-agent"}
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
		}).Debug("Systemctl executed")
	}
	return nil
}
