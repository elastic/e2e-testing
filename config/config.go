package config

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/elastic/metricbeat-tests-poc/docker"
	"github.com/elastic/metricbeat-tests-poc/log"
)

// OpWorkspace where the application works
var OpWorkspace string

// Op the tool's configuration, read from tool's workspace
var Op *opConfig

const fileName = "config.yml"

// checkInstalledSoftware checks that the required software is present
func checkInstalledSoftware() {
	log.Info("Validating required tools...")
	binaries := []string{
		"docker",
	}

	for _, binary := range binaries {
		which(binary)
	}
}

// Init creates this tool workspace under user's home, in a hidden directory named ".op"
func Init(services interface{}) {
	checkInstalledSoftware()

	usr, _ := user.Current()

	w := filepath.Join(usr.HomeDir, ".op")

	if _, err := os.Stat(w); os.IsNotExist(err) {
		err = os.MkdirAll(w, 0755)
		log.CheckIfErrorMessage(err, "Cannot create workdir for 'op' at "+w)

		log.Success("'op' workdir created at " + w)
	}

	OpWorkspace = w

	newConfig(w, services)

	docker.GetDevNetwork()
}

// opConfig tool configuration
type opConfig struct {
	Services map[string]interface{} `mapstructure:"services"`
}

// GetServiceConfig configuration of a service
func (c *opConfig) GetServiceConfig(service string) interface{} {
	return c.Services[service]
}

// newConfig returns a new configuration
func newConfig(workspace string, services interface{}) {
	opConfig, err := readConfig(workspace, fileName, map[string]interface{}{
		"services": services,
	})
	log.CheckIfErrorMessage(err, "Error when reading config.")

	Op = &opConfig
}

func initConfigFile(workspace string, configFile string, defaults map[string]interface{}) *os.File {
	log.Info("Creating %s with default values in %s.", configFile, workspace)

	configFilePath := filepath.Join(workspace, configFile)

	f, _ := os.Create(configFilePath)

	v := viper.New()
	for key, value := range defaults {
		v.SetDefault(key, value)
	}

	v.SetConfigType("yaml")
	v.SetConfigName("config")
	v.AddConfigPath(workspace)

	err := v.WriteConfig()
	log.CheckIfErrorMessage(err, `Cannot save default configuration file at `+configFilePath)

	return f
}

func readConfig(
	workspace string, configFile string, defaults map[string]interface{}) (opConfig, error) {

	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(workspace)

	err := viper.ReadInConfig()
	if err != nil {
		initConfigFile(workspace, configFile, defaults)
		viper.ReadInConfig()
	}

	var cfg opConfig
	err = viper.Unmarshal(&cfg)
	log.CheckIfErrorMessage(err, "Unable to decode configuration into struct")

	return cfg, err
}

// which checks if software is installed
func which(software string) {
	path, err := exec.LookPath(software)
	log.CheckIfError(err)
	log.Success("%s is present at %s", software, path)
}
