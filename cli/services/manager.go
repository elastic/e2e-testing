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
	RunCompose(bool, string, map[string]string) error
	StopCompose(bool, string) error
}

// DockerServiceManager implementation of the service manager interface
type DockerServiceManager struct {
}

// NewServiceManager returns a new service manager
func NewServiceManager() ServiceManager {
	return &DockerServiceManager{}
}

// RunCompose runs a docker compose by its name
func (sm *DockerServiceManager) RunCompose(
	isStack bool, composeName string, env map[string]string) error {

	composeFilePath, err := config.GetPackedCompose(isStack, composeName)
	if err != nil {
		return fmt.Errorf("Could not get compose file: %s - %v", composeFilePath, err)
	}
	defer os.Remove(composeFilePath)

	compose := tc.NewLocalDockerCompose([]string{composeFilePath}, composeName)
	execError := compose.
		WithCommand([]string{"up", "-d"}).
		WithEnv(env).
		Invoke()
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
