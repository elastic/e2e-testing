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
var Op *OpConfig

const fileName = "config.yml"

// servicesDefaults initial service configuration that could be overwritten by
// users on their local configuration. This configuration will be persisted in
// the application directory as initial configuration, in the form of a YAML file
var servicesDefaults = map[string]Service{
	"apache": {
		ContainerName: "apache-2.4",
		ExposedPort:   80,
		Image:         "httpd",
		Name:          "apache",
		NetworkAlias:  "apache",
		Version:       "2.4",
	},
	"elasticsearch": {
		BuildBranch:     "master",
		BuildRepository: "elastic/elasticsearch",
		ContainerName:   "elasticsearch-7.2.0",
		ExposedPort:     9200,
		Env: map[string]string{
			"bootstrap.memory_lock":  "true",
			"discovery.type":         "single-node",
			"ES_JAVA_OPTS":           "-Xms512m -Xmx512m",
			"xpack.security.enabled": "true",
		},
		Image:        "docker.elastic.co/elasticsearch/elasticsearch",
		Name:         "elasticsearch",
		NetworkAlias: "elasticsearch",
		Version:      "7.2.0",
	},
	"kafka": {
		ContainerName: "kafka",
		ExposedPort:   9092,
		Image:         "wurstmeister/kafka",
		Name:          "kafka",
		NetworkAlias:  "kafka",
		Version:       "latest",
	},
	"kibana": {
		BuildBranch:     "master",
		BuildRepository: "elastic/kibana",
		ContainerName:   "kibana-7.2.0",
		ExposedPort:     5601,
		Image:           "docker.elastic.co/kibana/kibana",
		Name:            "kibana",
		NetworkAlias:    "kibana",
		Version:         "7.2.0",
	},
	"metricbeat": {
		BuildBranch:     "master",
		BuildRepository: "elastic/beats",
		ContainerName:   "metricbeat-7.2.0",
		Image:           "docker.elastic.co/beats/metricbeat",
		Name:            "metricbeat",
		NetworkAlias:    "metricbeat",
		Version:         "7.2.0",
	},
	"mongodb": {
		ContainerName: "mongodb",
		ExposedPort:   27017,
		Image:         "mongo",
		Name:          "mongodb",
		NetworkAlias:  "mongodb",
		Version:       "latest",
	},
	"mysql": {
		ContainerName: "mysql",
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "secret",
		},
		ExposedPort:  3306,
		Image:        "mysql",
		Name:         "mysql",
		NetworkAlias: "mysql",
		Version:      "latest",
	},
	"redis": {
		ContainerName: "redis",
		ExposedPort:   6379,
		Image:         "redis",
		Name:          "redis",
		NetworkAlias:  "redis",
		Version:       "latest",
	},
}

// Service represents the configuration for a service
type Service struct {
	BindMounts      map[string]string `yaml:"BindMounts"`
	BuildBranch     string            `yaml:"BuildBranch"`
	BuildRepository string            `yaml:"BuildRepository"`
	ContainerName   string            `yaml:"ContainerName"`
	Daemon          bool              `yaml:"AsDaemon"`
	Env             map[string]string `yaml:"Env"`
	ExposedPort     int               `yaml:"ExposedPort"`
	Image           string            `yaml:"Image"`
	Labels          map[string]string `yaml:"Labels"`
	Name            string            `yaml:"Name"`
	NetworkAlias    string            `yaml:"NetworkAlias"`
	Version         string            `yaml:"Version"`
}

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
func Init() {
	checkInstalledSoftware()

	InitConfig()

	docker.GetDevNetwork()
}

// InitConfig initialises configuration
func InitConfig() {
	usr, _ := user.Current()

	w := filepath.Join(usr.HomeDir, ".op")

	if _, err := os.Stat(w); os.IsNotExist(err) {
		err = os.MkdirAll(w, 0755)
		log.CheckIfErrorMessage(err, "Cannot create workdir for 'op' at "+w)

		log.Success("'op' workdir created at " + w)

		initConfigFile(w)
	}

	OpWorkspace = w

	newConfig(w)
}

// OpConfig tool configuration
type OpConfig struct {
	Services map[string]Service `mapstructure:"services"`
}

// AvailableServices return the services in the configuration file
func AvailableServices() map[string]Service {
	return Op.Services
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

	opConfig, err := readConfig(workspace)
	log.CheckIfErrorMessage(err, "Error when reading config.")

	Op = &opConfig
}

func initConfigFile(workspace string) *os.File {
	log.Info("Creating %s with default values in %s.", fileName, workspace)

	configFilePath := filepath.Join(workspace, fileName)

	f, _ := os.Create(configFilePath)

	v := viper.New()
	for key, value := range servicesDefaults {
		v.SetDefault(key, value)
	}

	v.SetConfigType("yaml")
	v.SetConfigName("config")
	v.AddConfigPath(workspace)

	err := v.WriteConfig()
	log.CheckIfErrorMessage(err, `Cannot save default configuration file at `+configFilePath)
	log.Success("Config file initialised with default values")

	return f
}

func readConfig(workspace string) (OpConfig, error) {
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(workspace)

	err := viper.ReadInConfig()
	if err != nil {
		log.Warn("%v", err)
		initConfigFile(workspace)
		viper.ReadInConfig()
	}

	services := map[string]Service{}
	viper.Unmarshal(&services)
	log.CheckIfErrorMessage(err, "Unable to decode configuration into struct")

	cfg := OpConfig{
		Services: services,
	}

	return cfg, nil
}

// which checks if software is installed
func which(software string) {
	path, err := exec.LookPath(software)
	log.CheckIfError(err)
	log.Success("%s is present at %s", software, path)
}
