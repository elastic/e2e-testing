package config

import (
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/elastic/metricbeat-tests-poc/docker"
)

// OpWorkspace where the application works
var OpWorkspace string

// Op the tool's configuration, read from tool's workspace
var Op *OpConfig

const fileName = "config.yml"

// Init creates this tool workspace under user's home, in a hidden directory named ".op"
func Init(services interface{}) {
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
