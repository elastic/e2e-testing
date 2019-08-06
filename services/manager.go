package services

import (
	"fmt"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/elastic/metricbeat-tests-poc/config"
	"github.com/elastic/metricbeat-tests-poc/log"
)

// ServiceManager manages lifecycle of a service
type ServiceManager interface {
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

// Build builds a service domain entity from just its name and version
func (sm *DockerServiceManager) Build(service string, version string, asDaemon bool) Service {
	if service == "metricbeat" {
		return NewMetricbeatService(version, asDaemon)
	}

	cfg, exists := config.Op.GetServiceConfig(service)
	if !exists {
		log.Error("Cannot find service %s in configuration file.", service)
		return nil
	}

	srv := config.Service{}

	mapstructure.Decode(cfg, &srv)

	dockerService := DockerService{
		Service: srv,
	}
	dockerService.SetAsDaemon(asDaemon)
	dockerService.SetVersion(version)

	dockerService.SetContainerName(
		dockerService.GetName() + "-" + dockerService.GetVersion() + "-" + strconv.Itoa(int(time.Now().UnixNano())))

	return &dockerService
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
