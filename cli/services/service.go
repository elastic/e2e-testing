package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	config "github.com/elastic/metricbeat-tests-poc/cli/config"
	docker "github.com/elastic/metricbeat-tests-poc/cli/docker"
)

// Service represents the contract for services
type Service interface {
	CleanUp(context.Context) error
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
	SetCleanUp(func(context.Context) error)
	SetCmd(string)
	SetContainerName(string)
	SetEnv(map[string]string)
	SetLabels(map[string]string)
	SetNetworkAlias(string)
	SetVersion(string)
	SetWaitFor(wait.Strategy)
}

// DockerService represents a Docker service to be run
type DockerService struct {
	config.Service
	Cmd         string
	CleanUpFunc func(context.Context) error // function to execute when removing the container
	WaitFor     wait.Strategy
}

// CleanUp executes a function that performs clean up tasks
func (s *DockerService) CleanUp(ctx context.Context) error {
	if s.CleanUpFunc == nil {
		return nil
	}

	return s.CleanUpFunc(ctx)
}

// GetContainerName returns service name, which is calculated from service name and version
func (s *DockerService) GetContainerName() string {
	if s.ContainerName == "" {
		return s.Name + "-" + s.Version
	}

	return s.ContainerName
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

// SetCmd set the command to be executed on service startup
func (s *DockerService) SetCmd(cmd string) {
	s.Cmd = cmd
}

// SetContainerName set container name for a service
func (s *DockerService) SetContainerName(name string) {
	s.ContainerName = name
}

// SetBindMounts set bind mounts for a service
func (s *DockerService) SetBindMounts(bindMounts map[string]string) {
	s.BindMounts = bindMounts
}

// SetCleanUp set the clean up function to be executed when cleaning up the service
func (s *DockerService) SetCleanUp(fn func(ctx context.Context) error) {
	s.CleanUpFunc = fn
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

// SetNetworkAlias set network alias for a service
func (s *DockerService) SetNetworkAlias(alias string) {
	s.NetworkAlias = alias
}

// SetVersion set version for a service
func (s *DockerService) SetVersion(version string) {
	s.Version = version
}

// SetWaitFor set the Testcontainers' strategy to wait for
func (s *DockerService) SetWaitFor(strategy wait.Strategy) {
	s.WaitFor = strategy
}

func (s *DockerService) toLogFields(json *types.ContainerJSON) log.Fields {
	ip := json.NetworkSettings.IPAddress

	fields := log.Fields{
		"service":       s.GetName(),
		"image":         s.Image + ":" + s.GetVersion(),
		"network":       "elastic-dev-network",
		"containerName": s.GetContainerName(),
		"networkAlias":  s.GetNetworkAlias(),
		"IP":            ip,
		"Cmd":           s.Cmd,
	}

	i := 0
	for bm, path := range s.BindMounts {
		fields[fmt.Sprintf("bindMount_%d", i)] = fmt.Sprintf("%s=%s", bm, path)
		i++
	}

	i = 0
	for k, v := range s.Env {
		fields[fmt.Sprintf("envVar_%d", i)] = fmt.Sprintf("%s=%s", k, v)
		i++
	}

	i = 0
	for lb, label := range s.Labels {
		fields[fmt.Sprintf("label_%d", i)] = fmt.Sprintf("%s=%s", lb, label)
		i++
	}

	return fields
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
		log.WithFields(log.Fields{
			"service": s.GetName(),
		}).Info("The service is not present in the system")

		return nil
	}

	ctx := context.Background()

	err = docker.RemoveContainer(json.Name)
	if err == nil {
		s.CleanUp(ctx)

		log.WithFields(log.Fields{
			"service": s.GetName(),
		}).Debug("Service cleaned up")
	}

	return err
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
		Cmd:          s.Cmd,
		Env:          s.Env,
		ExposedPorts: exposedPorts,
		Labels:       s.Labels,
		Name:         s.ContainerName,
		SkipReaper:   s.Daemon,
		WaitingFor:   s.WaitFor,
	}

	service, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          false,
	})
	if err != nil {
		return nil, err
	}

	json, err := s.Inspect()
	if err != nil {
		return nil, err
	}

	docker.ConnectContainerToDevNetwork(json.ContainerJSONBase.ID, s.GetNetworkAlias())
	log.WithFields(log.Fields{
		"containerID":  json.ContainerJSONBase.ID,
		"networkAlias": s.GetNetworkAlias(),
	}).Debug("Service attached to Dev network")

	service.Start(ctx)

	log.WithFields(s.toLogFields(json)).Debug("Service information")
	log.WithFields(log.Fields{
		"containerName": s.GetContainerName(),
		"service":       s.GetName(),
		"version":       s.GetVersion(),
	}).Info("Service created")

	return service, nil
}

// AsDaemon marks this service to be run as daemon
func (s *DockerService) AsDaemon() *DockerService {
	s.Daemon = true

	return s
}
