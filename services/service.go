package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/testcontainers/testcontainers-go"

	config "github.com/elastic/metricbeat-tests-poc/config"
	docker "github.com/elastic/metricbeat-tests-poc/docker"
	"github.com/elastic/metricbeat-tests-poc/log"
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
	SetAsDaemon(bool)
	SetBindMounts(map[string]string)
	SetContainerName(string)
	SetEnv(map[string]string)
	SetLabels(map[string]string)
	SetVersion(string)
}

// DockerService represents a Docker service to be run
type DockerService struct {
	config.Service
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
	json, err := docker.InspectContainer(s.GetName() + "-" + s.GetVersion())
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

	if json == nil {
		log.Info("The service for %s is not present in the system", s.GetName())
		return nil
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

	if s.Labels == nil {
		s.Labels = map[string]string{}
	}

	s.Labels["service.owner"] = "co.elastic.observability"
	s.Labels["service.container.name"] = s.GetName() + "-" + s.GetVersion()

	s.SetContainerName(
		s.GetName() + "-" + s.GetVersion() + "-" + strconv.Itoa(int(time.Now().UnixNano())))

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
