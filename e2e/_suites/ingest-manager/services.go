package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// ElasticAgentInstaller represents how to install an agent, depending of the box type
type ElasticAgentInstaller struct {
	image         string // docker image
	InstallCmds   []string
	name          string // the name for the binary
	path          string // the path where the agent for the binary is installed
	profile       string // parent docker-compose file
	PostInstallFn func() error
	tag           string // docker tag
}

// NewCentosInstaller returns an instance of the Centos installer
func NewCentosInstaller() ElasticAgentInstaller {
	image := "centos"
	tag := "7"
	profile := "ingest-manager"

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	version := "8.0.0-SNAPSHOT"
	os := "linux"
	arch := "x86_64"
	extension := "tar.gz"

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
		image:         image,
		InstallCmds:   []string{"tar", "xzvf", tarFile},
		PostInstallFn: fn,
		profile:       profile,
		tag:           tag,
	}
}
