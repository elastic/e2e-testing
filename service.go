package main

import (
	"context"

	testcontainers "github.com/testcontainers/testcontainers-go"
)

// Service represents the contract for services
type Service interface {
	Destroy() error
	GetContainerName() string
	GetExposedPorts() []string
	GetName() string
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
	ImageTag       string
	Labels         map[string]string
	Name           string
	RunningService testcontainers.Container
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
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        s.ImageTag,
		BindMounts:   s.BindMounts,
		Env:          s.Env,
		ExposedPorts: s.GetExposedPorts(),
		Labels:       s.Labels,
		Name:         s.ContainerName,
	}

	service, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	s.RunningService = service

	return service, nil
}

// AsDaemon marks this service to be run as daemon
func (s *DockerService) AsDaemon() *DockerService {
	s.Daemon = true

	return s
}
