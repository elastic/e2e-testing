package services

import (
	"fmt"

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

	cfg := config.Op.GetServiceConfig(service)
	if cfg == nil {
		log.Error("Cannot find service %s in configuration file.", service)
		return nil
	}

	srv := &DockerService{}

	mapstructure.Decode(cfg, &srv)

	srv.SetAsDaemon(asDaemon)
	srv.SetVersion(version)

	srv.SetContainerName(srv.GetName() + "-" + srv.GetVersion())

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