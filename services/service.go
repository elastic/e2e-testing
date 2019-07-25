package services

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"

	docker "github.com/elastic/metricbeat-tests-poc/docker"
	testcontainers "github.com/testcontainers/testcontainers-go"
)

// Service represents the contract for services
type Service interface {
	Destroy() error
	GetContainerName() string
	GetExposedPorts() []string
	GetName() string
	GetVersion() string
	Inspect() (*types.ContainerJSON, error)
	Run() (testcontainers.Container, error)
}

// DockerService represents a Docker service to be run
type DockerService struct {
	BindMounts    map[string]string
	ContainerName string
	// Daemon indicates if the service must be run as a daemon
	Daemon         bool
	Env            map[string]string
	ExposedPorts   []ExposedPort
	Image          string
	Labels         map[string]string
	Name           string
	RunningService testcontainers.Container
	Version        string
}

// GetContainerName returns service name
func (s *DockerService) GetContainerName() string {
	return s.ContainerName
}

// GetExposedPorts returns an array of exposed ports
func (s *DockerService) GetExposedPorts() []string {
	ports := []string{}

	for _, p := range s.ExposedPorts {
		ports = append(ports, p.toString())
	}

	return ports
}

// GetName returns service name
func (s *DockerService) GetName() string {
	return s.Name
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

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        imageTag,
		BindMounts:   s.BindMounts,
		Env:          s.Env,
		ExposedPorts: s.GetExposedPorts(),
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
	Run(Service) error
}

// DockerServiceManager implementation of the service manager interface
type DockerServiceManager struct {
}

// NewServiceManager returns a new service manager
func NewServiceManager() ServiceManager {
	return &DockerServiceManager{}
}

// Run runs a service
func (sm *DockerServiceManager) Run(s Service) error {
	_, err := s.Run()
	if err != nil {
		return fmt.Errorf("Could not run service: %v", err)
	}

	return nil
}
