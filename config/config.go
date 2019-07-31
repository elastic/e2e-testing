package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// OpWorkspace where the application works
var OpWorkspace string

// Op the tool's configuration, read from tool's workspace
var Op *OpConfig

const fileName = "config.yml"

// OpConfig tool configuration
type OpConfig struct {
	Services map[string]interface{} `mapstructure:"services"`
}

// GetServiceConfig configuration of a service
func (c *OpConfig) GetServiceConfig(service string) interface{} {
	return c.Services[service]
}

// NewConfig returns a new configuration
func NewConfig(workspace string, services interface{}) {
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
