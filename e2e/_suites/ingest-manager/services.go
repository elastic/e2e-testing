package main

import (
	"fmt"

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
	image             string // docker image
	InstallCmds       []string
	name              string // the name for the binary
	path              string // the local path where the agent for the binary is located
	profile           string // parent docker-compose file
	PostInstallFn     func() error
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
	if "centos" == image {
		return newCentosInstaller()
	} else if "debian" == image {
		return newDebianInstaller()
	}

	log.WithField("image", image).Fatal("Sorry, we currently do not support this installer")
	return ElasticAgentInstaller{}
}

// newCentosInstaller returns an instance of the Centos installer
func newCentosInstaller() ElasticAgentInstaller {
	image := "centos"
	tag := "7"
	profile := "ingest-manager"

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	version := "8.0.0-SNAPSHOT"
	os := "linux"
	arch := "x86_64"
	extension := "tar.gz"

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

	extractedDir := fmt.Sprintf("%s-%s-%s-%s", artifact, version, os, arch)
	tarFile := fmt.Sprintf("%s.%s", extractedDir, extension)

	fn := func() error {
		// enable elastic-agent in PATH, because we know the location of the binary
		cmd := []string{"ln", "-s", "/" + extractedDir + "/elastic-agent", "/usr/local/bin/elastic-agent"}
		err := execCommandInService(profile, image, cmd, false)
		if err != nil {
			log.WithFields(log.Fields{
				"command": cmd,
				"error":   err,
				"service": image,
			}).Error("Could not symlink the agent to PATH")

			return err
		}

		log.WithFields(log.Fields{
			"command": cmd,
			"service": image,
		}).Debug("The symlink for the agent was created")
		return nil
	}

	return ElasticAgentInstaller{
		artifactArch:      arch,
		artifactExtension: extension,
		artifactName:      artifact,
		artifactOS:        os,
		artifactVersion:   version,
		image:             image,
		InstallCmds:       []string{"tar", "xzvf", tarFile},
		name:              binaryName,
		path:              binaryPath,
		PostInstallFn:     fn,
		profile:           profile,
		tag:               tag,
	}
}

// newDebianInstaller returns an instance of the Debian installer
func newDebianInstaller() ElasticAgentInstaller {
	image := "debian"
	tag := "9"
	profile := "ingest-manager"

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	version := "8.0.0-SNAPSHOT"
	os := "linux"
	arch := "x86_64"
	extension := "tar.gz"

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

	extractedDir := fmt.Sprintf("%s-%s-%s-%s", artifact, version, os, arch)
	tarFile := fmt.Sprintf("%s.%s", extractedDir, extension)

	fn := func() error {
		// enable elastic-agent in PATH, because we know the location of the binary
		cmd := []string{"ln", "-s", "/" + extractedDir + "/elastic-agent", "/usr/local/bin/elastic-agent"}
		err := execCommandInService(profile, image, cmd, false)
		if err != nil {
			log.WithFields(log.Fields{
				"command": cmd,
				"error":   err,
				"service": image,
			}).Error("Could not symlink the agent to PATH")

			return err
		}

		log.WithFields(log.Fields{
			"command": cmd,
			"service": image,
		}).Debug("The symlink for the agent was created")
		return nil
	}

	return ElasticAgentInstaller{
		artifactArch:      arch,
		artifactExtension: extension,
		artifactName:      artifact,
		artifactOS:        os,
		artifactVersion:   version,
		image:             image,
		InstallCmds:       []string{"tar", "xzvf", tarFile},
		name:              binaryName,
		path:              binaryPath,
		PostInstallFn:     fn,
		profile:           profile,
		tag:               tag,
	}
}
