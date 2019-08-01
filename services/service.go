package services

import (
	"context"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/mitchellh/mapstructure"
	"github.com/testcontainers/testcontainers-go"

	config "github.com/elastic/metricbeat-tests-poc/config"
	docker "github.com/elastic/metricbeat-tests-poc/docker"
	"github.com/elastic/metricbeat-tests-poc/log"
)

// servicesDefaults initial service configuration that could be overwritten by
// users on their local configuration. This configuration will be persisted in
// the application directory as initial configuration, in the form of a YAML file
var servicesDefaults = map[string]DockerService{
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
		ContainerName:   "mongodb",
		ExposedPort:     27017,
		Image:           "mongo",
		Name:            "mongodb",
		NetworkAlias:    "mongodb",
		Version:         "latest",
	},
	"mysql": {
		ContainerName:   "mysql",
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "secret",
		},
		ExposedPort:     3306,
		Image:           "mysql",
		Name:            "mysql",
		NetworkAlias:    "mysql",
		Version:         "latest",
	},
}

// Service represents the contract for services
type Service interface {
	Destroy() error
	GetContainerName() string
	GetExposedPort() string
	GetName() string
	GetNetworkAlias() string
	GetVersion() string
	Inspect() (*types.ContainerJSON, error)
	Run() (testcontainers.Container, error)
	SetAsDaemon(bool)
	SetBindMounts(map[string]string)
	SetContainerName(string)
	SetEnv(map[string]string)
	SetLabels(map[string]string)
	SetVersion(string)
}

// DockerService represents a Docker service to be run
type DockerService struct {
	BindMounts      map[string]string `yaml:"BindMounts"`
	BuildBranch     string            `yaml:"BuildBranch"`
	BuildRepository string            `yaml:"BuildRepository"`
	ContainerName   string            `yaml:"ContainerName"`
	// Daemon indicates if the service must be run as a daemon
	Daemon       bool              `yaml:"AsDaemon"`
	Env          map[string]string `yaml:"Env"`
	ExposedPort  int               `yaml:"ExposedPort"`
	Image        string            `yaml:"Image"`
	Labels       map[string]string `yaml:"Labels"`
	Name         string            `yaml:"Name"`
	NetworkAlias string            `yaml:"NetworkAlias"`
	Version      string            `yaml:"Version"`
}

// GetContainerName returns service name
func (s *DockerService) GetContainerName() string {
	return s.ContainerName
}

// GetExposedPort returns the string representation of a service's well-known exposed port
func (s *DockerService) GetExposedPort() string {
	return strconv.Itoa(s.ExposedPort)
}

// GetName returns service name
func (s *DockerService) GetName() string {
	return s.Name
}

// GetNetworkAlias returns service alias for the dev network
func (s *DockerService) GetNetworkAlias() string {
	if s.NetworkAlias == "" {
		return s.Name
	}

	return s.NetworkAlias
}

// GetVersion returns service name
func (s *DockerService) GetVersion() string {
	return s.Version
}

// Inspect returns the JSON representation of the container obtained from
// the Docker engine
func (s *DockerService) Inspect() (*types.ContainerJSON, error) {
	json, err := docker.InspectContainer(s.GetContainerName())
	if err != nil {
		return nil, fmt.Errorf("Could not inspect the container: %v", err)
	}

	return json, nil
}

// SetAsDaemon set if the service must be run as daemon
func (s *DockerService) SetAsDaemon(asDaemon bool) {
	s.Daemon = asDaemon
}

// SetContainerName set container name for a service
func (s *DockerService) SetContainerName(name string) {
	s.ContainerName = name
}

// SetBindMounts set bind mounts for a service
func (s *DockerService) SetBindMounts(bindMounts map[string]string) {
	s.BindMounts = bindMounts
}

// SetEnv set environment variables for a service
func (s *DockerService) SetEnv(env map[string]string) {
	s.Env = env
}

// SetLabels set labels for a service
func (s *DockerService) SetLabels(labels map[string]string) {
	s.Labels = labels
}

// SetVersion set version for a service
func (s *DockerService) SetVersion(version string) {
	s.Version = version
}

// ExposedPort represents the structure for how services expose ports
type ExposedPort struct {
	Address       string
	ContainerPort string
	HostPort      string
	Protocol      string
}

func (e *ExposedPort) toString() string {
	return e.Address + ":" + e.HostPort + ":" + e.ContainerPort + "/" + e.Protocol
}

// Destroy destroys the underlying container
func (s *DockerService) Destroy() error {
	json, err := s.Inspect()
	if err != nil {
		return err
	}

	return docker.RemoveContainer(json.Name)
}

// Run runs a container for the service
func (s *DockerService) Run() (testcontainers.Container, error) {
	imageTag := s.Image + ":" + s.Version

	exposedPorts := []string{}

	if s.ExposedPort != 0 {
		exposedPort := ExposedPort{
			Address:       "0.0.0.0",
			ContainerPort: s.GetExposedPort(),
			Protocol:      "tcp",
		}

		exposedPorts = append(exposedPorts, exposedPort.toString())
	}

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        imageTag,
		BindMounts:   s.BindMounts,
		Env:          s.Env,
		ExposedPorts: exposedPorts,
		Labels:       s.Labels,
		Name:         s.ContainerName,
		SkipReaper:   !s.Daemon,
	}

	service, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	json, err := s.Inspect()
	if err != nil {
		return nil, err
	}

	docker.ConnectContainerToDevNetwork(json.ContainerJSONBase.ID, s.GetNetworkAlias())

	ip := json.NetworkSettings.IPAddress
	ports := json.NetworkSettings.Ports
	log.Info("The service (%s) runs on %s %v", s.GetName(), ip, ports)

	return service, nil
}

// AsDaemon marks this service to be run as daemon
func (s *DockerService) AsDaemon() *DockerService {
	s.Daemon = true

	return s
}

// ServiceManager manages lifecycle of a service
type ServiceManager interface {
	AvailableServices() map[string]DockerService
	Build(string, string, bool) Service
	Run(Service) error
	Stop(Service) error
}

// DockerServiceManager implementation of the service manager interface
type DockerServiceManager struct {
}

// NewServiceManager returns a new service manager
func NewServiceManager() ServiceManager {
	return &DockerServiceManager{}
}

// AvailableServices returns the available services in the system
func (sm *DockerServiceManager) AvailableServices() map[string]DockerService {
	return servicesDefaults
}

// Build builds a service domain entity from just its name and version
func (sm *DockerServiceManager) Build(service string, version string, asDaemon bool) Service {
	if service == "kibana" {
		return NewKibanaService(version, asDaemon)
	} else if service == "metricbeat" {
		return NewMetricbeatService(version, asDaemon)
	}

	cfg := config.Op.GetServiceConfig(service)
	if cfg == nil {
		log.Error("Cannot find service %s in configuration file.", service)
		return nil
	}

	srv := &DockerService{}

	mapstructure.Decode(cfg, &srv)

	srv.SetAsDaemon(asDaemon)
	srv.SetVersion(version)

	srv.SetContainerName(srv.GetName()+"-"+srv.GetVersion())

	return srv
}

// Run runs a service
func (sm *DockerServiceManager) Run(s Service) error {
	_, err := s.Run()
	if err != nil {
		return fmt.Errorf("Could not run service: %v", err)
	}

	return nil
}

// Stop stops a service
func (sm *DockerServiceManager) Stop(s Service) error {
	err := s.Destroy()
	if err != nil {
		return err
	}

	return nil
}
