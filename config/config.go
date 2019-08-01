package config

import (
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/elastic/metricbeat-tests-poc/docker"
	logger "github.com/elastic/metricbeat-tests-poc/log"
)

// OpWorkspace where the application works
var OpWorkspace string

// Op the tool's configuration, read from tool's workspace
var Op *OpConfig

const fileName = "config.yml"

// checkInstalledSoftware checks that the required software is present
func checkInstalledSoftware() {
	logger.Info("Validating required tools...")
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
		if err != nil {
			log.Fatalf("Cannot create workdir for 'op' at "+w, err)
		}

		log.Println("'op' workdir created at " + w)
	}

	OpWorkspace = w

	newConfig(w, services)

	docker.GetDevNetwork()
}

// OpConfig tool configuration
type OpConfig struct {
	Services map[string]interface{} `mapstructure:"services"`
}

// GetServiceConfig configuration of a service
func (c *OpConfig) GetServiceConfig(service string) interface{} {
	return c.Services[service]
}

// newConfig returns a new configuration
func newConfig(workspace string, services interface{}) {
	opConfig, err := readConfig(workspace, fileName, map[string]interface{}{
		"services": services,
	})
	if err != nil {
		log.Fatalf("Error when reading config: %v\n", err)
	}

	Op = &opConfig
}

func initConfigFile(workspace string, configFile string, defaults map[string]interface{}) *os.File {
	log.Printf("Creating %s with default values in %s.", configFile, workspace)

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
	if err != nil {
		log.Fatalf(`Cannot save default configuration file at %s: %v`, configFilePath, err)
	}

	return f
}

func readConfig(
	workspace string, configFile string, defaults map[string]interface{}) (OpConfig, error) {

	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(workspace)

	err := viper.ReadInConfig()
	if err != nil {
		initConfigFile(workspace, configFile, defaults)
		viper.ReadInConfig()
	}

	var cfg OpConfig
	err = viper.Unmarshal(&cfg)
	if err != nil {
		log.Fatalf("Unable to decode configuration into struct, %v", err)
	}

	return cfg, err
}

// which checks if software is installed
func which(software string) {
	path, err := exec.LookPath(software)
	logger.CheckIfError(err)
	logger.Success("%s is present at %s", software, path)
}
