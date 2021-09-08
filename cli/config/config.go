// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/e2e-testing/internal/git"
	io "github.com/elastic/e2e-testing/internal/io"
	"github.com/elastic/e2e-testing/internal/sanitizer"
	shell "github.com/elastic/e2e-testing/internal/shell"

	packr "github.com/gobuffalo/packr/v2"
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

// Op the tool's configuration, read from tool's workspace
var Op *OpConfig

// OpConfig tool configuration
type OpConfig struct {
	Profiles  map[string]Profile `mapstructure:"profiles"`
	Services  map[string]Service `mapstructure:"services"`
	workspace string             `mapstructure:"workspace"`
}

// GetServiceConfig configuration of a service
func (c *OpConfig) GetServiceConfig(service string) (Service, bool) {
	srv, exists := c.Services[service]

	return srv, exists
}

// Service represents the configuration for a service
type Service struct {
	Name string `mapstructure:"Name"`
	Path string `mapstructure:"Path"`
}

// Profile represents the configuration for an aggregation of services
// represented by a Docker Compose
type Profile struct {
	Name string `mapstructure:"Name"`
	Path string `mapstructure:"Path"`
}

// AvailableServices return the services in the configuration file
func AvailableServices() map[string]Service {
	return Op.Services
}

// AvailableProfiles return the profiles in the configuration file
func AvailableProfiles() map[string]Profile {
	return Op.Profiles
}

// FileExists checks if a configuration file exists
func FileExists(configFile string) (bool, error) {
	return io.Exists(configFile)
}

// GetServiceConfig configuration of a service
func GetServiceConfig(service string) (Service, bool) {
	return Op.GetServiceConfig(service)
}

// Init creates this tool workspace under user's home, in a hidden directory named ".op"
func Init() {
	if Op != nil {
		return
	}

	configureLogger()

	// Remote provider does not require the use of docker
	if shell.GetEnv("PROVIDER", "docker") != "remote" {
		binaries := []string{
			"docker",
			"docker-compose",
		}
		shell.CheckInstalledSoftware(binaries...)
	}

	home, err := homedir.Dir()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Could not get current user's HOME dir")
	}

	workspace := filepath.Join(home, ".op")

	newConfig(workspace)
}

// OpDir returns the directory to copy to
func OpDir() string {
	return Op.workspace
}

// PutServiceEnvironment puts the environment variables for the service, replacing "SERVICE_"
// with service name in uppercase. The variables are:
//  - SERVICE_VERSION: where it represents the version of the service (i.e. APACHE_VERSION)
//  - SERVICE_PATH: where it represents the path to its compose file (i.e. APACHE_PATH)
func PutServiceEnvironment(env map[string]string, service string, serviceVersion string) map[string]string {
	serviceUpper := strings.ToUpper(service)

	if _, exists := env[serviceUpper+"_VERSION"]; !exists {
		env[serviceUpper+"_VERSION"] = serviceVersion
	}

	srv, exists := Op.Services[service]
	if !exists {
		log.WithFields(log.Fields{
			"service": service,
		}).Warn("Could not find compose file")
	} else {
		env[serviceUpper+"_PATH"] = filepath.Dir(srv.Path)
	}

	return env
}

// PutServiceVariantEnvironment puts the environment variables that comes in the supported-versions.yml
// file of the service, replacing "SERVICE_ with service name in uppercase. At the end, it also adds
// the version and the path for the service, calling the PutServiceEnvironment method. An example:
//  - SERVICE_VARIANT: where SERVICE is the name of the service (i.e. APACHE_VARIANT)
//  - SERVICE_VERSION: where it represents the version of the service (i.e. APACHE_VERSION)
func PutServiceVariantEnvironment(env map[string]string, service string, serviceVariant string, serviceVersion string) map[string]string {
	type EnvVar interface{}
	type supportedVersions struct {
		Env []EnvVar `yaml:"variants"`
	}

	versionsPath := filepath.Join(
		OpDir(), "compose", "services", service, "_meta", "supported-versions.yml")

	bytes, err := io.ReadFile(versionsPath)
	if err != nil {
		return map[string]string{}
	}

	sv := supportedVersions{}
	err = yaml.Unmarshal(bytes, &sv)
	if err != nil {
		log.WithFields(log.Fields{
			"supported-versions": versionsPath,
		}).Error("Could not unmarshal supported versions")

		return map[string]string{}
	}

	// discover variants and set them only if the variant matches
	for _, e := range sv.Env {
		switch i := e.(type) {
		case map[interface{}]interface{}:
			for k, v := range i {
				srvVariant := fmt.Sprint(v)
				if srvVariant == serviceVariant {
					env[fmt.Sprint(k)] = srvVariant
				}
			}
		default:
			// skip
		}
	}

	return PutServiceEnvironment(env, service, serviceVersion)
}

// SyncIntegrations clones Beats repository, extracting descriptors for the supported metricbeat's integration modules
func SyncIntegrations(deleteRepository bool, remote string) error {
	workspace := OpDir()

	// BeatsRepo default object representing Beats project
	var BeatsRepo = git.ProjectBuilder.
		WithBaseWorkspace(filepath.Join(workspace, "git")).
		WithDomain("github.com").
		WithName("beats").
		WithRemote(remote).
		Build()

	if deleteRepository {
		repoDir := filepath.Join(workspace, "git", BeatsRepo.Name)

		log.WithFields(log.Fields{
			"path": repoDir,
		}).Trace("Removing repository")
		os.RemoveAll(repoDir)
		log.WithFields(log.Fields{
			"path": repoDir,
		}).Trace("Repository removed")
	}

	git.Clone(BeatsRepo)

	pattern := filepath.Join(
		BeatsRepo.GetWorkspace(), "metricbeat", "module", "*", "_meta", "supported-versions.yml")
	copyIntegrationsComposeFiles(BeatsRepo, pattern, workspace)

	xPackPattern := filepath.Join(
		BeatsRepo.GetWorkspace(), "x-pack", "metricbeat", "module", "*", "_meta", "supported-versions.yml")
	copyIntegrationsComposeFiles(BeatsRepo, xPackPattern, workspace)

	return nil
}

func checkConfigDirectory(dir string) {
	found, err := io.Exists(dir)
	if found && err == nil {
		return
	}
	_ = io.MkdirAll(dir)
}

func checkConfigDirs(workspace string) {
	servicesPath := filepath.Join(workspace, "compose", "services")
	profilesPath := filepath.Join(workspace, "compose", "profiles")

	checkConfigDirectory(servicesPath)
	checkConfigDirectory(profilesPath)

	log.WithFields(log.Fields{
		"servicesPath": servicesPath,
		"profilesPath": profilesPath,
	}).Trace("'op' workdirs created.")
}

func configureLogger() {
	includeTimestamp := os.Getenv("OP_LOG_INCLUDE_TIMESTAMP")
	fullTimestamp := (strings.ToUpper(includeTimestamp) == "TRUE")

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: fullTimestamp,
	})

	switch logLevel := os.Getenv("OP_LOG_LEVEL"); logLevel {
	case "TRACE":
		log.SetLevel(log.TraceLevel)
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "WARNING":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	case "FATAL":
		log.SetLevel(log.FatalLevel)
	case "PANIC":
		log.SetLevel(log.PanicLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

// tells whether the a array contains the x string.
func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// CopyComposeFiles copies only those services that has a supported-versions.yml
// file from Beats integrations, and we will need to copy them into a directory
// named as the original service (i.e. aerospike) under this tool's workspace,
// alongside the services. Besides that, the method will copy the _meta directory
// for each service, also sanitising the compose files: it will remove the 'build'
// blocks from the compose files.
func copyIntegrationsComposeFiles(beats git.Project, pattern string, target string) {
	files := io.FindFiles(pattern)

	for _, file := range files {
		metaDir := filepath.Dir(file)
		serviceDir := filepath.Dir(metaDir)
		service := filepath.Base(serviceDir)

		composeFile := filepath.Join(serviceDir, "docker-compose.yml")
		configFile := filepath.Join(serviceDir, "_meta", "config.yml")
		targetFile := filepath.Join(
			target, "compose", "services", service, "docker-compose.yml")
		targetConfigFile := filepath.Join(
			target, "compose", "services", service, "_meta", "config.yml")

		err := io.MkdirAll(filepath.Dir(targetFile))
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  file,
			}).Warn("File was not copied")
			continue
		}

		err = io.CopyFile(composeFile, targetFile, 10000)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  composeFile,
			}).Warn("File was not copied")
			continue
		}

		err = io.CopyFile(configFile, targetConfigFile, 10000)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  configFile,
			}).Warn("File was not copied")
			continue
		}

		targetMetaDir := filepath.Join(target, "compose", "services", service, "_meta")
		err = io.CopyDir(metaDir, targetMetaDir)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"_meta": metaDir,
			}).Warn("Meta dir was not copied")
			continue
		}
		log.WithFields(log.Fields{
			"meta":       metaDir,
			"targetMeta": targetMetaDir,
		}).Trace("Integration _meta directory copied")

		err = sanitizeComposeFile(targetFile, targetFile)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  targetFile,
			}).Warn("Could not sanitize compose file")
			continue
		}
		log.WithFields(log.Fields{
			"file":       file,
			"targetFile": targetFile,
		}).Trace("Integration compose file copied")

		err = sanitizeConfigurationFile(service, targetConfigFile)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  targetFile,
			}).Warn("Could not sanitize config file")
			continue
		}
		log.WithFields(log.Fields{
			"file":       file,
			"targetFile": targetConfigFile,
		}).Trace("Integration config file copied")
	}

	log.WithFields(log.Fields{
		"files":  len(files),
		"target": target,
	}).Info("Integrations files copied")
}

type service interface{}
type composeFile struct {
	Version  string             `yaml:"version"`
	Services map[string]service `yaml:"services"`
}

// newConfig returns a new configuration
func newConfig(workspace string) {
	if Op != nil {
		return
	}

	opConfig := OpConfig{
		Services:  map[string]Service{},
		Profiles:  map[string]Profile{},
		workspace: workspace,
	}
	Op = &opConfig

	checkConfigDirs(Op.workspace)

	box := packFiles(Op)
	if box == nil {
		log.WithFields(log.Fields{
			"workspace": workspace,
		}).Error("Could not get packaged compose files")
		return
	}

	// initialize included profiles/services
	extractProfileServiceConfig(Op, box)

	// add file system services and profiles
	readFilesFromFileSystem("services")
	readFilesFromFileSystem("profiles")
}

// Extract packaged profiles and services for use with cli runner
// all default configs/profiles/services will be overwritten. In order to customize
// the default deployments, create a new directory and copy the existing files over.
func extractProfileServiceConfig(op *OpConfig, box *packr.Box) error {
	var walkFn = func(s string, file packr.File) error {
		p := filepath.Join(OpDir(), "compose", s)
		dir := filepath.Dir(p)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				return err
			}
		}

		log.WithFields(log.Fields{
			"file": p,
			"dir":  dir,
		}).Trace("Extracting boxed file")

		return ioutil.WriteFile(p, []byte(file.String()), 0644)
	}

	if err := box.Walk(walkFn); err != nil {
		return err
	}
	return nil
}

// Packs all files required for starting profiles/services
// if a configurations directory exists within the parent directory for each compose file
// it will be included in the package binary but skipped in determining the correct paths
// for each compose file
//
// Directory format for profiles/services are as follows:
// compose/
//  profiles/
//    fleet/
//      configurations/
//      docker-compose.yml
//  services/
//    apache/
//      configurations/
//      docker-compose.yml
//
// configurations/ directory is optional and only needed if docker-compose.yml needs to reference
// any filelike object within its parent directory
func packFiles(op *OpConfig) *packr.Box {
	box := packr.New("Compose Files", "compose")
	return box
}

// reads the docker-compose in the workspace, merging them with what it's
// already boxed in the binary
func readFilesFromFileSystem(serviceType string) {
	basePath := filepath.Join(OpDir(), "compose", serviceType)
	files, err := io.ReadDir(basePath)
	if err != nil {
		log.WithFields(log.Fields{
			"path": basePath,
			"type": serviceType,
		}).Warn("Could not load file system")
		return
	}

	for _, f := range files {
		if f.IsDir() {
			name := f.Name()
			composeFilePath := filepath.Join(basePath, name, "docker-compose.yml")
			found, err := io.Exists(composeFilePath)
			if found && err == nil {
				log.WithFields(log.Fields{
					"service": name,
					"path":    composeFilePath,
				}).Trace("workspace file")

				if serviceType == "services" {
					// add a service or a profile
					Op.Services[name] = Service{
						Name: f.Name(),
						Path: composeFilePath,
					}
				} else if serviceType == "profiles" {
					Op.Profiles[name] = Profile{
						Name: f.Name(),
						Path: composeFilePath,
					}
				}
			}
		}
	}
}

// removes non-needed blocks in the target compose file, such as the build context
func sanitizeComposeFile(composeFilePath string, targetFilePath string) error {
	bytes, err := io.ReadFile(composeFilePath)
	if err != nil {
		log.WithFields(log.Fields{
			"docker-compose": composeFilePath,
		}).Error("Could not read docker compose file")
		return err
	}

	c := composeFile{}
	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		log.WithFields(log.Fields{
			"docker-compose": composeFilePath,
		}).Error("Could not unmarshal docker compose file")
		return err
	}

	// sets version to 2.4 for testing purpose
	c.Version = "2.4"

	var excludedBlocks = []string{"build"}

	for k, srv := range c.Services {
		// we'll copy all fields but the excluded ones in this struct
		output := make(map[string]service)

		switch i := srv.(type) {
		case map[interface{}]interface{}:
			log.WithFields(log.Fields{
				"name":         k,
				"compose-file": composeFilePath,
			}).Trace("sanitize service in docker-compose file")

			for key, value := range i {
				strKey := fmt.Sprintf("%v", key)

				// remove the build context element
				if contains(excludedBlocks, strKey) {
					continue
				}

				output[strKey] = value
			}
		default:
			// skip
		}

		c.Services[k] = output
	}

	d, err := yaml.Marshal(&c)
	if err != nil {
		return err
	}

	return io.WriteFile(d, targetFilePath)
}

// sanitizeConfigurationFile replaces 127.0.0.1 with current service name
func sanitizeConfigurationFile(serviceName string, configFilePath string) error {
	bytes, err := io.ReadFile(configFilePath)
	if err != nil {
		log.WithFields(log.Fields{
			"config.yml": configFilePath,
		}).Error("Could not read config.yml file")
		return err
	}

	// replace local IP with service name, because it will be discovered under Docker Compose network
	content := strings.ReplaceAll(string(bytes), "127.0.0.1", serviceName)
	content = strings.ReplaceAll(content, "localhost", serviceName)
	// prepend modules header
	content = "metricbeat.modules:\n" + content

	serviceSanitizer := sanitizer.GetConfigSanitizer(serviceName)
	content = serviceSanitizer.Sanitize(content)

	log.WithFields(log.Fields{
		"config.reference.yml": configFilePath,
	}).Trace("Config file sanitized")

	return io.WriteFile([]byte(content), configFilePath)
}
