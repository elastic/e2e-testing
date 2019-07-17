package main

import (
	"context"

	testcontainers "github.com/testcontainers/testcontainers-go"
)

// Service represents a service to be run
type Service struct {
	// Daemon indicates if the service must be run as a daemon
	Daemon         bool
	ExposedPorts   []ExposedPort
	ImageTag       string
	RunningService testcontainers.Container
}

func (s *Service) exposePorts() []string {
	ports := []string{}

	for _, p := range s.ExposedPorts {
		ports = append(ports, p.toString())
	}

	return ports
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

// destroys the underlying container
func (s *Service) destroy() error {
	ctx := context.Background()

	s.RunningService.Terminate(ctx)

	return nil
}

// runs a container for the service
func (s *Service) run() (testcontainers.Container, error) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        s.ImageTag,
		ExposedPorts: s.exposePorts(),
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
