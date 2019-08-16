package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"

	config "github.com/elastic/metricbeat-tests-poc/cli/config"
	docker "github.com/elastic/metricbeat-tests-poc/cli/docker"
	"github.com/elastic/metricbeat-tests-poc/cli/log"
)

// Service represents the contract for services
type Service interface {
	Destroy() error
	GetContainerName() string
	GetExposedPort(int) string
	GetExposedPorts() []int
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

// GetContainerName returns service name, which is calculated from service name and version
func (s *DockerService) GetContainerName() string {
	return s.Name + "-" + s.Version
}

// GetExposedPort returns the string representation of a service's well-known exposed ports
func (s *DockerService) GetExposedPort(i int) string {
	return strconv.Itoa(s.ExposedPorts[i])
}

// GetExposedPorts returns a service's well-known exposed ports
func (s *DockerService) GetExposedPorts() []int {
	return s.ExposedPorts
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

// SetEnv set environment variables for a service, overriding default one with
// those defined in the configuration file
func (s *DockerService) SetEnv(env map[string]string) {
	if s.Env == nil {
		s.Env = map[string]string{}
	}

	for k, v := range env {
		s.Env[k] = v
	}
}

// SetLabels set labels for a service
func (s *DockerService) SetLabels(labels map[string]string) {
	s.Labels = labels
}

// SetVersion set version for a service
func (s *DockerService) SetVersion(version string) {
	s.Version = version
}

func (s *DockerService) toString(json *types.ContainerJSON) string {
	ip := json.NetworkSettings.IPAddress
	ports := json.NetworkSettings.Ports

	toString := ""
	toString += fmt.Sprintf("\tService: %s\n", s.GetName())
	toString += fmt.Sprintf("\tImage : %s:%s\n", s.Image, s.GetVersion())
	toString += fmt.Sprintf("\tNetwork: %v\n", "elastic-dev-network")
	toString += fmt.Sprintf("\tContainer Name: %s\n", s.GetContainerName())
	toString += fmt.Sprintf("\tNetwork Alias: %s\n", s.GetNetworkAlias())
	toString += fmt.Sprintf("\tIP: %s\n", ip)
	toString += fmt.Sprintf("\tApplication Ports\n")
	for _, port := range s.ExposedPorts {
		sPort := fmt.Sprintf("%d/tcp", port)
		toString += fmt.Sprintf("\t\t%d -> %v\n", port, ports[nat.Port(sPort)])
	}
	toString += fmt.Sprintf("\tBind Mounts:\n")
	for bm, path := range s.BindMounts {
		toString += fmt.Sprintf("\t\t%s -> %s\n", bm, path)
	}
	toString += fmt.Sprintf("\tEnvironment Variables:\n")
	for k, v := range s.Env {
		toString += fmt.Sprintf("\t\t%s : %s\n", k, v)
	}
	toString += fmt.Sprintf("\tLabels:\n")
	for lb, label := range s.Labels {
		toString += fmt.Sprintf("\t\t%s -> %s\n", lb, label)
	}

	return toString
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

	for i := range s.ExposedPorts {
		exposedPort := ExposedPort{
			Address:       "0.0.0.0",
			ContainerPort: s.GetExposedPort(i),
			Protocol:      "tcp",
		}

		exposedPorts = append(exposedPorts, exposedPort.toString())
	}

	if s.Labels == nil {
		s.Labels = map[string]string{}
	}

	s.Labels["service.owner"] = "co.elastic.observability"
	s.Labels["service.container.name"] = s.GetName() + "-" + s.GetVersion()

	s.SetContainerName(s.GetContainerName() + "-" + strconv.Itoa(int(time.Now().UnixNano())))

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

	log.Success("The service (%s) has been created successfully:", s.GetName())
	log.Log("%s", s.toString(json))

	return service, nil
}

// AsDaemon marks this service to be run as daemon
func (s *DockerService) AsDaemon() *DockerService {
	s.Daemon = true

	return s
}
