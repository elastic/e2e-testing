// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	io "github.com/elastic/e2e-testing/cli/internal"
	shell "github.com/elastic/e2e-testing/cli/shell"

	packr "github.com/gobuffalo/packr/v2"
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

// opComposeBox the tool's static files where we will embed default Docker compose
// files representing the services and the profiles
var opComposeBox *packr.Box

// Op the tool's configuration, read from tool's workspace
var Op *OpConfig

// OpConfig tool configuration
type OpConfig struct {
	Profiles  map[string]Profile `mapstructure:"profiles"`
	Services  map[string]Service `mapstructure:"services"`
	Workspace string             `mapstructure:"workspace"`
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

// GetComposeFile returns the path of the compose file, looking up the
// tool's workdir or in the static resources already packaged in the binary
func GetComposeFile(isProfile bool, composeName string) (string, error) {
	composeFileName := "docker-compose.yml"
	serviceType := "services"
	if isProfile {
		serviceType = "profiles"
	}

	composeFilePath := path.Join(Op.Workspace, "compose", serviceType, composeName, composeFileName)
	found, err := io.Exists(composeFilePath)
	if found && err == nil {
		log.WithFields(log.Fields{
			"composeFilePath": composeFilePath,
			"type":            serviceType,
		}).Debug("Compose file found at workdir")

		return composeFilePath, nil
	}

	log.WithFields(log.Fields{
		"composeFilePath": composeFilePath,
		"error":           err,
		"type":            serviceType,
	}).Debug("Compose file not found at workdir. Extracting from binary resources")

	composeBytes, err := opComposeBox.Find(path.Join(serviceType, composeName, composeFileName))
	if err != nil {
		log.WithFields(log.Fields{
			"composeFileName": composeFileName,
			"error":           err,
			"isProfile":       isProfile,
			"type":            serviceType,
		}).Error("Could not find compose file.")

		return "", err
	}

	// create parent directory for the compose file
	err = io.MkdirAll(filepath.Dir(composeFilePath))
	if err != nil {
		return composeFilePath, err
	}

	err = io.WriteFile(composeBytes, composeFilePath)
	if err != nil {
		return composeFilePath, err
	}

	log.WithFields(log.Fields{
		"composeFilePath": composeFilePath,
		"isProfile":       isProfile,
		"type":            serviceType,
	}).Debug("Compose file generated at workdir.")

	return composeFilePath, nil
}

// GetServiceConfig configuration of a service
func GetServiceConfig(service string) (Service, bool) {
	return Op.GetServiceConfig(service)
}

// Init creates this tool workspace under user's home, in a hidden directory named ".op"
func Init() {
	configureLogger()

	binaries := []string{
		"docker",
		"docker-compose",
	}
	shell.CheckInstalledSoftware(binaries)

	InitConfig()
}

// InitConfig initialises configuration
func InitConfig() {
	if Op != nil {
		return
	}

	home, err := homedir.Dir()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Could not get current user's HOME dir")
	}

	w := filepath.Join(home, ".op")

	newConfig(w)
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

	versionsPath := path.Join(
		Op.Workspace, "compose", "services", service, "_meta", "supported-versions.yml")

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

func checkConfigDirectory(dir string) {
	found, err := io.Exists(dir)
	if found && err == nil {
		return
	}
	_ = io.MkdirAll(dir)
}

func checkConfigDirs(workspace string) {
	servicesPath := path.Join(workspace, "compose", "services")
	profilesPath := path.Join(workspace, "compose", "profiles")

	checkConfigDirectory(servicesPath)
	checkConfigDirectory(profilesPath)

	log.WithFields(log.Fields{
		"servicesPath": servicesPath,
		"profilesPath": profilesPath,
	}).Debug("'op' workdirs created.")
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

// newConfig returns a new configuration
func newConfig(workspace string) {
	if Op != nil {
		return
	}

	checkConfigDirs(workspace)

	opConfig := OpConfig{
		Services:  map[string]Service{},
		Profiles:  map[string]Profile{},
		Workspace: workspace,
	}
	Op = &opConfig

	box := packComposeFiles(Op)
	if box == nil {
		log.WithFields(log.Fields{
			"workspace": workspace,
		}).Error("Could not get packaged compose files")
		return
	}

	// add file system services and profiles
	readFilesFromFileSystem("services")
	readFilesFromFileSystem("profiles")

	opComposeBox = box
}

func packComposeFiles(op *OpConfig) *packr.Box {
	box := packr.New("Compose Files", "./compose")

	err := box.Walk(func(boxedPath string, f packr.File) error {
		// there must be three tokens: i.e. 'services/aerospike/docker-compose.yml'
		tokens := strings.Split(boxedPath, string(os.PathSeparator))
		composeType := tokens[0]
		composeName := tokens[1]

		log.WithFields(log.Fields{
			"service": composeName,
			"path":    boxedPath,
		}).Debug("Boxed file")

		if composeType == "profiles" {
			op.Profiles[composeName] = Profile{
				Name: composeName,
				Path: boxedPath,
			}
		} else if composeType == "services" {
			op.Services[composeName] = Service{
				Name: composeName,
				Path: boxedPath,
			}
		}

		return nil
	})
	if err != nil {
		return nil
	}

	return box
}

// reads the docker-compose in the workspace, merging them with what it's
// already boxed in the binary
func readFilesFromFileSystem(serviceType string) {
	basePath := path.Join(Op.Workspace, "compose", serviceType)
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
				}).Debug("Workspace file")

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
