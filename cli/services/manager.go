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
func (sm *DockerServiceManager) AddServicesToCompose(stack string, composeNames []string, env map[string]string) error {
	log.WithFields(log.Fields{
		"stack":    stack,
		"services": composeNames,
	}).Debug("Adding services to compose")

	newComposeNames := []string{stack}
	newComposeNames = append(newComposeNames, composeNames...)

	return executeCompose(sm, false, newComposeNames, []string{"up", "-d"}, map[string]string{})
}

// RemoveServicesFromCompose removes services from a running docker compose
func (sm *DockerServiceManager) RemoveServicesFromCompose(stack string, composeNames []string) error {
	log.WithFields(log.Fields{
		"stack":    stack,
		"services": composeNames,
	}).Debug("Removing services to compose")

	newComposeNames := []string{stack}
	newComposeNames = append(newComposeNames, composeNames...)

	command := []string{"kill"}
	command = append(command, composeNames...)

	return executeCompose(sm, false, newComposeNames, command, map[string]string{})
}

// RunCompose runs a docker compose by its name
func (sm *DockerServiceManager) RunCompose(isStack bool, composeNames []string, env map[string]string) error {
	return executeCompose(sm, isStack, composeNames, []string{"up", "-d"}, env)
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

func executeCompose(sm *DockerServiceManager, isStack bool, composeNames []string, command []string, env map[string]string) error {
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
		WithCommand(command).
		WithEnv(env).
		Invoke()
	err := execError.Error
	if err != nil {
		return fmt.Errorf("Could not run compose file: %v - %v", composeFilePaths, err)
	}

	log.WithFields(log.Fields{
		"cmd":              command,
		"composeFilePaths": composeFilePaths,
		"env":              env,
		"stack":            composeNames[0],
	}).Debug("Docker compose executed.")

	return nil
}
