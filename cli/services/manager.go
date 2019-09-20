package services

import (
	"fmt"
	"os"

	"github.com/elastic/metricbeat-tests-poc/cli/config"

	log "github.com/sirupsen/logrus"
	tc "github.com/testcontainers/testcontainers-go"
)

// ServiceManager manages lifecycle of a service
type ServiceManager interface {
	Build(string, string, bool) Service
	BuildFromConfig(config.Service) Service
	Run(Service) error
	RunCompose(bool, string) error
	Stop(Service) error
	StopCompose(bool, string) error
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
	cfg, exists := config.GetServiceConfig(service)
	if !exists {
		log.WithFields(log.Fields{
			"service": service,
			"version": version,
			"daemon":  asDaemon,
		}).Fatal("Cannot find service in configuration.")
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

// RunCompose runs a docker compose by its name
func (sm *DockerServiceManager) RunCompose(isStack bool, composeName string) error {
	composeFilePath, err := config.GetPackedCompose(isStack, composeName)
	if err != nil {
		return fmt.Errorf("Could not get compose file: %s - %v", composeFilePath, err)
	}
	defer os.Remove(composeFilePath)

	compose := tc.NewLocalDockerCompose([]string{composeFilePath}, composeName)
	execError := compose.WithCommand([]string{"up", "-d"}).Invoke()
	err = execError.Error
	if err != nil {
		return fmt.Errorf("Could not run compose file: %s - %v", composeFilePath, err)
	}

	log.WithFields(log.Fields{
		"composeFilePath": composeFilePath,
		"stack":           composeName,
	}).Debug("Docker compose up.")

	return nil
}

// StopCompose stops a docker compose by its name
func (sm *DockerServiceManager) StopCompose(isStack bool, composeName string) error {
	composeFilePath, err := config.GetPackedCompose(isStack, composeName)
	if err != nil {
		return fmt.Errorf("Could not get compose file: %s - %v", composeFilePath, err)
	}
	defer os.Remove(composeFilePath)

	compose := tc.NewLocalDockerCompose([]string{composeFilePath}, composeName)
	execError := compose.Down()
	err = execError.Error
	if err != nil {
		return fmt.Errorf("Could not stop compose file: %s - %v", composeFilePath, err)
	}

	log.WithFields(log.Fields{
		"composeFilePath": composeFilePath,
		"stack":           composeName,
	}).Debug("Docker compose down.")

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
