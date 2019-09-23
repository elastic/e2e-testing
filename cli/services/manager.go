package services

import (
	"fmt"

	"github.com/elastic/metricbeat-tests-poc/cli/config"

	log "github.com/sirupsen/logrus"
	tc "github.com/testcontainers/testcontainers-go"
)

// ServiceManager manages lifecycle of a service
type ServiceManager interface {
	AddServicesToCompose(string, []string, map[string]string) error
	RemoveServicesFromCompose(string, []string) error
	RunCompose(bool, []string, map[string]string) error
	StopCompose(bool, []string) error
}

// DockerServiceManager implementation of the service manager interface
type DockerServiceManager struct {
}

// NewServiceManager returns a new service manager
func NewServiceManager() ServiceManager {
	return &DockerServiceManager{}
}

// AddServicesToCompose adds services to a running docker compose
func (sm *DockerServiceManager) AddServicesToCompose(
	stack string, composeNames []string, env map[string]string) error {

	log.WithFields(log.Fields{
		"stack":    stack,
		"services": composeNames,
	}).Debug("Adding services to compose")

	// add services to the running stack
	composeFilePaths := make([]string, len(composeNames)+1)

	composeFilePath, err := config.GetPackedCompose(true, stack)
	if err != nil {
		return fmt.Errorf("Could not get compose file for the stack: %s - %v", composeFilePath, err)
	}
	// first compose file is the one relative to the stack
	composeFilePaths[0] = composeFilePath

	for i, composeName := range composeNames {
		composeFilePath, err := config.GetPackedCompose(false, composeName)
		if err != nil {
			return fmt.Errorf("Could not get compose file: %s - %v", composeFilePath, err)
		}
		composeFilePaths[i+1] = composeFilePath
	}

	compose := tc.NewLocalDockerCompose(composeFilePaths, stack)
	execError := compose.
		WithCommand([]string{"up", "-d"}).
		WithEnv(env).
		Invoke()
	err = execError.Error
	if err != nil {
		return fmt.Errorf("Could not run compose file: %v - %v", composeFilePaths, err)
	}

	log.WithFields(log.Fields{
		"composeFilePaths": composeFilePaths,
		"stack":            composeNames[0],
	}).Debug("Services added to the stack.")

	return nil
}

// RemoveServicesFromCompose removes services from a running docker compose
func (sm *DockerServiceManager) RemoveServicesFromCompose(stack string, composeNames []string) error {
	log.WithFields(log.Fields{
		"stack":    stack,
		"services": composeNames,
	}).Debug("Removing services to compose")

	// add services to the running stack
	composeFilePaths := make([]string, len(composeNames)+1)

	composeFilePath, err := config.GetPackedCompose(true, stack)
	if err != nil {
		return fmt.Errorf("Could not get compose file for the stack: %s - %v", composeFilePath, err)
	}
	// first compose file is the one relative to the stack
	composeFilePaths[0] = composeFilePath

	for i, composeName := range composeNames {
		composeFilePath, err := config.GetPackedCompose(false, composeName)
		if err != nil {
			return fmt.Errorf("Could not get compose file: %s - %v", composeFilePath, err)
		}
		composeFilePaths[i+1] = composeFilePath
	}

	command := []string{"kill"}
	command = append(command, composeNames...)

	compose := tc.NewLocalDockerCompose(composeFilePaths, stack)
	execError := compose.
		WithCommand(command).
		Invoke()
	err = execError.Error
	if err != nil {
		return fmt.Errorf("Could not run compose file: %v - %v", composeFilePaths, err)
	}

	log.WithFields(log.Fields{
		"composeFilePaths": composeFilePaths,
		"stack":            composeNames[0],
	}).Debug("Services removed from the stack.")

	return nil
}

// RunCompose runs a docker compose by its name
func (sm *DockerServiceManager) RunCompose(
	isStack bool, composeNames []string, env map[string]string) error {

	composeFilePaths := make([]string, len(composeNames))
	for i, composeName := range composeNames {
		composeFilePath, err := config.GetPackedCompose(isStack, composeName)
		if err != nil {
			return fmt.Errorf("Could not get compose file: %s - %v", composeFilePath, err)
		}
		composeFilePaths[i] = composeFilePath
	}

	compose := tc.NewLocalDockerCompose(composeFilePaths, composeNames[0])
	execError := compose.
		WithCommand([]string{"up", "-d"}).
		WithEnv(env).
		Invoke()
	err := execError.Error
	if err != nil {
		return fmt.Errorf("Could not run compose file: %v - %v", composeFilePaths, err)
	}

	log.WithFields(log.Fields{
		"composeFilePaths": composeFilePaths,
		"stack":            composeNames[0],
	}).Debug("Docker compose up.")

	return nil
}

// StopCompose stops a docker compose by its name
func (sm *DockerServiceManager) StopCompose(isStack bool, composeNames []string) error {
	composeFilePaths := make([]string, len(composeNames))
	for i, composeName := range composeNames {
		composeFilePath, err := config.GetPackedCompose(isStack, composeName)
		if err != nil {
			return fmt.Errorf("Could not get compose file: %s - %v", composeFilePath, err)
		}
		composeFilePaths[i] = composeFilePath
	}

	compose := tc.NewLocalDockerCompose(composeFilePaths, composeNames[0])
	execError := compose.Down()
	err := execError.Error
	if err != nil {
		return fmt.Errorf("Could not stop compose file: %v - %v", composeFilePaths, err)
	}

	log.WithFields(log.Fields{
		"composeFilePath": composeFilePaths,
		"stack":           composeNames[0],
	}).Debug("Docker compose down.")

	return nil
}
