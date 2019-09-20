package config

import (
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	packr "github.com/gobuffalo/packr/v2"
	log "github.com/sirupsen/logrus"

	"github.com/elastic/metricbeat-tests-poc/cli/docker"
)

// OpComposeBox the tool's static files where we will embed default Docker compose
// files representing the services and the stacks
var OpComposeBox *packr.Box

// Op the tool's configuration, read from tool's workspace
var Op *OpConfig

const fileName = "config.yml"

// servicesDefaults initial service configuration
var servicesDefaults = map[string]Service{}

// Service represents the configuration for a service
type Service struct {
	Name string `mapstructure:"Name"`
	Path string `mapstructure:"Path"`
}

var stacksDefaults = map[string]Stack{}

// Stack represents the configuration for a stack, which is an aggregation of services
// represented by a Docker Compose
type Stack struct {
	Name string `mapstructure:"Name"`
	Path string `mapstructure:"Path"`
}

// checkInstalledSoftware checks that the required software is present
func checkInstalledSoftware() {
	log.Debug("Validating required tools...")
	binaries := []string{
		"docker",
	}

	for _, binary := range binaries {
		which(binary)
	}
}

// Init creates this tool workspace under user's home, in a hidden directory named ".op"
func Init() {
	configureLogger()

	checkInstalledSoftware()

	InitConfig()

	docker.GetDevNetwork()
}

// InitConfig initialises configuration
func InitConfig() {
	if Op != nil {
		return
	}

	usr, _ := user.Current()

	w := filepath.Join(usr.HomeDir, ".op")

	newConfig(w)
}

// OpConfig tool configuration
type OpConfig struct {
	Services map[string]Service `mapstructure:"services"`
	Stacks   map[string]Stack   `mapstructure:"stacks"`
}

// AvailableServices return the services in the configuration file
func AvailableServices() map[string]Service {
	return Op.Services
}

// AvailableStacks return the stacks in the configuration file
func AvailableStacks() map[string]Stack {
	return Op.Stacks
}

// GetServiceConfig configuration of a service
func GetServiceConfig(service string) (Service, bool) {
	return Op.GetServiceConfig(service)
}

// GetServiceConfig configuration of a service
func (c *OpConfig) GetServiceConfig(service string) (Service, bool) {
	srv, exists := c.Services[service]

	return srv, exists
}

// newConfig returns a new configuration
func newConfig(workspace string) {
	if Op != nil {
		return
	}

	opConfig := OpConfig{
		Services: map[string]Service{},
		Stacks:   map[string]Stack{},
	}

	Op = &opConfig
	OpComposeBox = packComposeFiles(Op)
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

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// which checks if software is installed
func which(software string) {
	path, err := exec.LookPath(software)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"software": software,
		}).Fatal("Required binary is not present")
	}

	log.WithFields(log.Fields{
		"software": software,
		"path":     path,
	}).Debug("Binary is present")
}

func packComposeFiles(op *OpConfig) *packr.Box {
	box := packr.New("Compose Files", "./compose")

	box.Walk(func(boxedPath string, f packr.File) error {
		log.WithFields(log.Fields{
			"path": boxedPath,
		}).Debug("Boxed file")

		composeType, composeNameWithExtension := path.Split(boxedPath)
		composeName := strings.TrimSuffix(composeNameWithExtension, filepath.Ext(composeNameWithExtension))
		if composeType == "stacks/" {
			op.Stacks[composeName] = Stack{
				Name: composeName,
				Path: boxedPath,
			}
		} else if composeType == "services/" {
			op.Services[composeName] = Service{
				Name: composeName,
				Path: boxedPath,
			}
		}

		return nil
	})

	return box
}

// GetPackedCompose returns the path of the compose file, looking up the
// static resources already packaged in the binary
func GetPackedCompose(isStack bool, composeName string) (string, error) {
	composeFileName := composeName + ".yml"
	serviceType := "services"
	if isStack {
		serviceType = "stacks"
	}

	tmp, err := ioutil.TempDir("", "op")
	if err != nil {
		log.WithFields(log.Fields{
			"compose": composeName,
			"error":   err,
			"isStack": isStack,
			"type":    serviceType,
		}).Warn("Cannot create temporary directory. Trying locally")

		tmp = "."
	}

	composeBytes, _ := OpComposeBox.Find(path.Join(serviceType, composeFileName))

	composeFilePath := path.Join(tmp, composeFileName)
	err = ioutil.WriteFile(composeFilePath, composeBytes, 0755)
	if err != nil {
		log.WithFields(log.Fields{
			"composeFilePath": composeFilePath,
			"error":           err,
			"isStack":         isStack,
			"type":            serviceType,
		}).Error("Cannot write file.")

		return composeFilePath, err
	}

	log.WithFields(log.Fields{
		"composeFilePath": composeFilePath,
		"isStack":         isStack,
		"type":            serviceType,
	}).Debug("Temporary compose file generated.")

	return composeFilePath, nil
}
