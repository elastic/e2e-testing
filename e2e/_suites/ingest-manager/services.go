package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/cli/shell"
	"github.com/elastic/e2e-testing/e2e"
	log "github.com/sirupsen/logrus"
)

const agentVersionBase = "7.9.2"

// agentVersion is the version of the agent to use
// It can be overriden by ELASTIC_AGENT_VERSION env var
var agentVersion = "7.9.2"

func init() {
	config.Init()

	agentVersion = shell.GetEnv("ELASTIC_AGENT_VERSION", agentVersionBase)
}

// ElasticAgentInstaller represents how to install an agent, depending of the box type
type ElasticAgentInstaller struct {
	artifactArch      string // architecture of the artifact
	artifactExtension string // extension of the artifact
	artifactName      string // name of the artifact
	artifactOS        string // OS of the artifact
	artifactVersion   string // version of the artifact
	commitFile        string // elastic agent commit file
	homeDir           string // elastic agent home dir
	image             string // docker image
	InstallCmds       []string
	logDir            string // location of the log file
	logFile           string // the name of the log file
	name              string // the name for the binary
	path              string // the local path where the agent for the binary is located
	processName       string // name of the elastic-agent process
	profile           string // parent docker-compose file
	PostInstallFn     func() error
	service           string // name of the service
	tag               string // docker tag
}

// getElasticAgentLogs uses elastic-agent log dir to read the entire log file
func (i *ElasticAgentInstaller) getElasticAgentLogs(hostname string) error {
	containerName := hostname // name of the container, which matches the hostname

	logFile := i.logDir + i.logFile
	cmd := []string{
		"cat", logFile,
	}

	err := execCommandInService(i.profile, i.image, i.service, cmd, false)
	if err != nil {
		log.WithFields(log.Fields{
			"containerName": containerName,
			"command":       cmd,
			"error":         err,
		}).Error("Could not get agent logs in the container")

		return err
	}

	return nil
}

// downloadAgentBinary it downloads the binary and stores the location of the downloaded file
// into the installer struct, to be used else where
// If the environment variable ELASTIC_AGENT_DOWNLOAD_URL exists, then the artifact to be downloaded will
// be defined by that value
// Else, if the environment variable ELASTIC_AGENT_USE_CI_SNAPSHOTS is set, then the artifact
// to be downloaded will be defined by the latest snapshot produced by the Beats CI.
func downloadAgentBinary(artifact string, version string, OS string, arch string, extension string) (string, string, error) {
	fileName := fmt.Sprintf("%s-%s-%s.%s", artifact, version, arch, extension)

	if downloadURL, exists := os.LookupEnv("ELASTIC_AGENT_DOWNLOAD_URL"); exists {
		filePath, err := e2e.DownloadFile(downloadURL)

		return fileName, filePath, err
	}

	var downloadURL string
	var err error

	useCISnapshots, _ := shell.GetEnvBool("ELASTIC_AGENT_USE_CI_SNAPSHOTS")
	if useCISnapshots {
		log.Debug("Using CI snapshots for the Elastic Agent")

		// We will use the snapshots produced by Beats CI
		bucket := "beats-ci-artifacts"
		object := fmt.Sprintf("snapshots/%s", fileName)

		// we are setting a version from a pull request: the version of the artifact will be kept as the base one
		// i.e. /pull-requests/pr-21100/elastic-agent/elastic-agent-7.9.2-x86_64.rpm
		// i.e. /pull-requests/pr-21100/elastic-agent/elastic-agent-7.9.2-amd64.deb
		if strings.HasPrefix(version, "pr-") {
			fileName = fmt.Sprintf("%s-%s-%s.%s", artifact, agentVersionBase, arch, extension)
			log.WithFields(log.Fields{
				"agentVersion": agentVersionBase,
				"PR":           version,
			}).Debug("Using CI snapshots for a pull request")
			object = fmt.Sprintf("pull-requests/%s/%s/%s", version, artifact, fileName)
		}

		downloadURL, err = e2e.GetObjectURLFromBucket(bucket, object)
		if err != nil {
			return "", "", err
		}

		filePath, err := e2e.DownloadFile(downloadURL)

		return fileName, filePath, err
	}

	downloadURL, err = e2e.GetElasticArtifactURL(artifact, agentVersionBase, OS, arch, extension)
	if err != nil {
		return "", "", err
	}

	filePath, err := e2e.DownloadFile(downloadURL)

	return fileName, filePath, err
}

// GetElasticAgentInstaller returns an installer from a docker image
func GetElasticAgentInstaller(image string) ElasticAgentInstaller {
	log.WithFields(log.Fields{
		"image": image,
	}).Debug("Configuring installer for the agent")

	var installer ElasticAgentInstaller
	var err error
	if "centos-systemd" == image {
		installer, err = newCentosInstaller("centos-systemd", "latest")
	} else if "debian-systemd" == image {
		installer, err = newDebianInstaller()
	} else {
		log.WithField("image", image).Fatal("Sorry, we currently do not support this installer")
		return ElasticAgentInstaller{}
	}

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"image": image,
		}).Fatal("Sorry, we could not download the installer")
	}
	return installer
}

// newCentosInstaller returns an instance of the Centos installer
func newCentosInstaller(image string, tag string) (ElasticAgentInstaller, error) {
	service := image
	profile := IngestManagerProfileName

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
		return ElasticAgentInstaller{}, err
	}

	fn := func() error {
		return systemctlRun(profile, image, service, "enable")
	}

	return ElasticAgentInstaller{
		artifactArch:      arch,
		artifactExtension: extension,
		artifactName:      artifact,
		artifactOS:        os,
		artifactVersion:   version,
		commitFile:        ".elastic-agent.active.commit",
		homeDir:           "/etc/elastic-agent/",
		image:             image,
		InstallCmds:       []string{"yum", "localinstall", "/" + binaryName, "-y"},
		logDir:            "/var/lib/elastic-agent/logs/",
		logFile:           "elastic-agent-json.log",
		name:              binaryName,
		path:              binaryPath,
		PostInstallFn:     fn,
		processName:       ElasticAgentProcessName,
		profile:           profile,
		service:           service,
		tag:               tag,
	}, nil
}

// newDebianInstaller returns an instance of the Debian installer
func newDebianInstaller() (ElasticAgentInstaller, error) {
	image := "debian-systemd"
	service := image
	tag := "stretch"
	profile := IngestManagerProfileName

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
		return ElasticAgentInstaller{}, err
	}

	fn := func() error {
		return systemctlRun(profile, image, service, "enable")
	}

	return ElasticAgentInstaller{
		artifactArch:      arch,
		artifactExtension: extension,
		artifactName:      artifact,
		artifactOS:        os,
		artifactVersion:   version,
		commitFile:        ".elastic-agent.active.commit",
		homeDir:           "/etc/elastic-agent/",
		image:             image,
		InstallCmds:       []string{"apt", "install", "/" + binaryName, "-y"},
		logDir:            "/var/lib/elastic-agent/logs/",
		logFile:           "elastic-agent-json.log",
		name:              binaryName,
		path:              binaryPath,
		PostInstallFn:     fn,
		processName:       ElasticAgentProcessName,
		profile:           profile,
		service:           service,
		tag:               tag,
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
