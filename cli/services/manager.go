package services

import (
	"fmt"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
	"github.com/elastic/metricbeat-tests-poc/cli/log"
)

// ServiceManager manages lifecycle of a service
type ServiceManager interface {
	Build(string, string, bool) Service
	BuildFromConfig(config.Service) Service
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

	cfg, exists := config.GetServiceConfig(service)
	if !exists {
		log.Error("Cannot find service %s in configuration file.", service)
	}

	cfg.Daemon = asDaemon
	cfg.Version = version

	return sm.BuildFromConfig(cfg)
}

// BuildFromConfig builds a service domain entity from its configuration
func (sm *DockerServiceManager) BuildFromConfig(service config.Service) Service {
	dockerService := DockerService{
		Service: service,
	}

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
