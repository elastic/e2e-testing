package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/elastic/e2e-testing/cli/docker"
	"github.com/elastic/e2e-testing/cli/shell"
	"github.com/elastic/e2e-testing/e2e"
	log "github.com/sirupsen/logrus"
)

// to avoid downloading the same artifacts, we are adding this map to cache the URL of the downloaded binaries, using as key
// the URL of the artifact. If another installer is trying to download the same URL, it will return the location of the
// already downloaded artifact.
var binariesCache = map[string]string{}

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
func downloadAgentBinary(artifact string, version string, OS string, arch string, extension string, stale bool) (string, string, error) {
	fileName := fmt.Sprintf("%s-%s-%s.%s", artifact, version, arch, extension)

	beatsLocalPath := shell.GetEnv("BEATS_LOCAL_PATH", "")
	if beatsLocalPath != "" {
		distributions := path.Join(beatsLocalPath, "x-pack", "elastic-agent", "build", "distributions")
		log.Debugf("Using local snapshots for the Elastic Agent: %s", distributions)

		if extension == "tar.gz" {
			fileName = fmt.Sprintf("%s-%s-%s-%s.%s", artifact, version, OS, arch, extension)
		}

		fileNamePath := path.Join(distributions, fileName)
		_, err := os.Stat(fileNamePath)
		if err != nil || os.IsNotExist(err) {
			return fileName, fileNamePath, err
		}

		return fileName, fileNamePath, err
	}

	handleDownload := func(URL string, fileName string) (string, string, error) {
		if val, ok := binariesCache[URL]; ok {
			log.WithFields(log.Fields{
				"URL":  URL,
				"path": val,
			}).Debug("Retrieving binary from local cache")
			return fileName, val, nil
		}

		filePath, err := e2e.DownloadFile(URL)
		if err != nil {
			return fileName, filePath, err
		}

		binariesCache[URL] = filePath

		return fileName, filePath, nil
	}

	if downloadURL, exists := os.LookupEnv("ELASTIC_AGENT_DOWNLOAD_URL"); exists {
		return handleDownload(downloadURL, fileName)
	}

	var downloadURL string
	var err error

	useCISnapshots := shell.GetEnvBool("BEATS_USE_CI_SNAPSHOTS")
	if useCISnapshots {
		log.Debug("Using CI snapshots for the Elastic Agent")

		bucketFileName, bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, OS, arch, extension, stale)

		maxTimeout := time.Duration(timeoutFactor) * time.Minute

		downloadURL, err = e2e.GetObjectURLFromBucket(bucket, prefix, object, maxTimeout)
		if err != nil {
			return "", "", err
		}

		return handleDownload(downloadURL, bucketFileName)
	}

	downloadVersion := version
	if !stale {
		downloadVersion = checkElasticAgentVersion(version)
	}

	downloadURL, err = e2e.GetElasticArtifactURL(artifact, downloadVersion, OS, arch, extension)
	if err != nil {
		return "", "", err
	}

	return handleDownload(downloadURL, fileName)
}

// GetElasticAgentInstaller returns an installer from a docker image
func GetElasticAgentInstaller(image string, installerType string, version string, stale bool) ElasticAgentInstaller {
	log.WithFields(log.Fields{
		"image":     image,
		"installer": installerType,
	}).Debug("Configuring installer for the agent")

	var installer ElasticAgentInstaller
	var err error
	if "centos" == image && "tar" == installerType {
		installer, err = newTarInstaller("centos", "latest", version, stale)
	} else if "centos" == image && "systemd" == installerType {
		installer, err = newCentosInstaller("centos", "latest", version, stale)
	} else if "debian" == image && "tar" == installerType {
		installer, err = newTarInstaller("debian", "stretch", version, stale)
	} else if "debian" == image && "systemd" == installerType {
		installer, err = newDebianInstaller("debian", "stretch", version, stale)
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

// getGCPBucketCoordinates it calculates the bucket path in GCP
func getGCPBucketCoordinates(fileName string, artifact string, version string, OS string, arch string, extension string, stale bool) (string, string, string, string) {
	if extension == "tar.gz" {
		fileName = fmt.Sprintf("%s-%s-%s-%s.%s", artifact, version, OS, arch, extension)
	}

	bucket := "beats-ci-artifacts"
	prefix := fmt.Sprintf("snapshots/%s", artifact)
	object := fileName
	newFileName := fileName

	// the commit SHA will identify univocally the artifact in the GCP storage bucket
	commitSHA := shell.GetEnv("GITHUB_CHECK_SHA1", "")
	if commitSHA != "" {
		prefix = fmt.Sprintf("commits/%s", commitSHA)
		object = artifact + "/" + fileName
	}

	// we are setting a version from a pull request: the version of the artifact will be kept as the base one
	// i.e. /pull-requests/pr-21100/elastic-agent/elastic-agent-8.0.0-SNAPSHOT-x86_64.rpm
	// i.e. /pull-requests/pr-21100/elastic-agent/elastic-agent-8.0.0-SNAPSHOT-amd64.deb
	// i.e. /pull-requests/pr-21100/elastic-agent/elastic-agent-8.0.0-SNAPSHOT-linux-x86_64.tar.gz
	if strings.HasPrefix(strings.ToLower(version), "pr-") {
		newFileName = fmt.Sprintf("%s-%s-%s.%s", artifact, agentVersionBase, arch, extension)
		if extension == "tar.gz" {
			newFileName = fmt.Sprintf("%s-%s-%s-%s.%s", artifact, agentVersionBase, OS, arch, extension)
		}
		log.WithFields(log.Fields{
			"agentVersion": agentVersionBase,
			"PR":           version,
		}).Debug("Using CI snapshots for a pull request")
		prefix = fmt.Sprintf("pull-requests/%s", version)
		object = fmt.Sprintf("%s/%s", artifact, newFileName)
	}

	if stale {
		prefix = fmt.Sprintf("snapshots/%s", artifact)
		object = newFileName
	}

	return newFileName, bucket, prefix, object
}

func isSystemdBased(image string) bool {
	return strings.HasSuffix(image, "-systemd")
}

// newCentosInstaller returns an instance of the Centos installer for a specific version
func newCentosInstaller(image string, tag string, version string, stale bool) (ElasticAgentInstaller, error) {
	image = image + "-systemd" // we want to consume systemd boxes
	service := image
	profile := FleetProfileName

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	os := "linux"
	arch := "x86_64"
	extension := "rpm"

	binaryName, binaryPath, err := downloadAgentBinary(artifact, version, os, arch, extension, stale)
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
func newDebianInstaller(image string, tag string, version string, stale bool) (ElasticAgentInstaller, error) {
	image = image + "-systemd" // we want to consume systemd boxes
	service := image
	profile := FleetProfileName

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	os := "linux"
	arch := "amd64"
	extension := "deb"

	binaryName, binaryPath, err := downloadAgentBinary(artifact, version, os, arch, extension, stale)
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

// newTarInstaller returns an instance of the Debian installer for a specific version
func newTarInstaller(image string, tag string, version string, stale bool) (ElasticAgentInstaller, error) {
	image = image + "-systemd" // we want to consume systemd boxes
	service := image
	profile := FleetProfileName

	// extract the agent in the box, as it's mounted as a volume
	artifact := "elastic-agent"
	os := "linux"
	arch := "x86_64"
	extension := "tar.gz"

	tarFile, binaryPath, err := downloadAgentBinary(artifact, version, os, arch, extension, stale)
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
	installerPackage := NewTARPackage(tarFile, profile, image, service).
		Stale(stale).
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
		name:              tarFile,
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
