package services

import (
	"context"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types"

	docker "github.com/elastic/metricbeat-tests-poc/docker"
	testcontainers "github.com/testcontainers/testcontainers-go"
)

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
	SetBindMounts(map[string]string)
	SetEnv(map[string]string)
	SetLabels(map[string]string)
}

// DockerService represents a Docker service to be run
type DockerService struct {
	BindMounts    map[string]string
	ContainerName string
	// Daemon indicates if the service must be run as a daemon
	Daemon         bool
	Env            map[string]string
	ExposedPort    int
	Image          string
	Labels         map[string]string
	Name           string
	NetworkAlias   string
	RunningService testcontainers.Container
	Version        string
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
	ctx := context.Background()

	s.RunningService.Terminate(ctx)

	return nil
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

	s.RunningService = service

	json, err := s.Inspect()
	if err != nil {
		return nil, err
	}

	docker.ConnectContainerToDevNetwork(json.ContainerJSONBase.ID, s.GetNetworkAlias())

	ip := json.NetworkSettings.IPAddress
	ports := json.NetworkSettings.Ports
	fmt.Printf("The service (%s) runs on %s %v\n", s.GetName(), ip, ports)

	return service, nil
}

// AsDaemon marks this service to be run as daemon
func (s *DockerService) AsDaemon() *DockerService {
	s.Daemon = true

	return s
}

// ServiceManager manages lifecycle of a service
type ServiceManager interface {
	Build(string, string) Service
	Run(Service) error
}

// DockerServiceManager implementation of the service manager interface
type DockerServiceManager struct {
}

// NewServiceManager returns a new service manager
func NewServiceManager() ServiceManager {
	return &DockerServiceManager{}
}

// Build builds a service domain entity from just its name and version
func (sm *DockerServiceManager) Build(service string, version string) Service {
	if service == "apache" {
		return NewApacheService(version, true)
	} else if service == "kafka" {
		return NewKafkaService(version, true)
	} else if service == "metricbeat" {
		return NewMetricbeatService(version, true)
	} else if service == "mysql" {
		return NewMySQLService(version, true)
	}

	return nil
}

// Run runs a service
func (sm *DockerServiceManager) Run(s Service) error {
	_, err := s.Run()
	if err != nil {
		return fmt.Errorf("Could not run service: %v", err)
	}

	return nil
}
