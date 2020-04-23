package services

import (
	"fmt"
	"path/filepath"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
	state "github.com/elastic/metricbeat-tests-poc/cli/internal"

	log "github.com/sirupsen/logrus"
	tc "github.com/testcontainers/testcontainers-go"
)

// ServiceManager manages lifecycle of a service
type ServiceManager interface {
	AddServicesToCompose(stack string, composeNames []string, env map[string]string) error
	RemoveServicesFromCompose(stack string, composeNames []string, env map[string]string) error
	RunCommand(stack string, composeNames []string, composeArgs []string, env map[string]string) error
	RunCompose(isStack bool, composeNames []string, env map[string]string) error
	StopCompose(isStack bool, composeNames []string) error
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

	persistedEnv := state.Recover(stack+"-stack", config.Op.Workspace)
	for k, v := range env {
		persistedEnv[k] = v
	}

	err := executeCompose(sm, true, newComposeNames, []string{"up", "-d"}, persistedEnv)
	if err != nil {
		return err
	}

	return nil
}

// RemoveServicesFromCompose removes services from a running docker compose
func (sm *DockerServiceManager) RemoveServicesFromCompose(stack string, composeNames []string, env map[string]string) error {
	log.WithFields(log.Fields{
		"stack":    stack,
		"services": composeNames,
	}).Debug("Removing services to compose")

	newComposeNames := []string{stack}
	newComposeNames = append(newComposeNames, composeNames...)

	persistedEnv := state.Recover(stack+"-stack", config.Op.Workspace)
	for k, v := range env {
		persistedEnv[k] = v
	}

	for _, composeName := range composeNames {
		command := []string{"rm", "-fvs"}
		command = append(command, composeName)

		err := executeCompose(sm, true, newComposeNames, command, persistedEnv)
		if err != nil {
			log.WithFields(log.Fields{
				"command":  command,
				"services": composeNames,
				"stack":    stack,
			}).Error("Could not remove services")
			return err
		}
	}

	return nil
}

// RunCommand executes a docker-compose command in a running a docker compose
func (sm *DockerServiceManager) RunCommand(stack string, composeNames []string, composeArgs []string, env map[string]string) error {
	return executeCompose(sm, true, composeNames, composeArgs, env)
}

// RunCompose runs a docker compose by its name
func (sm *DockerServiceManager) RunCompose(isStack bool, composeNames []string, env map[string]string) error {
	return executeCompose(sm, isStack, composeNames, []string{"up", "-d"}, env)
}

// StopCompose stops a docker compose by its name
func (sm *DockerServiceManager) StopCompose(isStack bool, composeNames []string) error {
	composeFilePaths := make([]string, len(composeNames))
	for i, composeName := range composeNames {
		b := isStack
		if i == 0 && !isStack && (len(composeName) == 1) {
			b = true
		}

		composeFilePath, err := config.GetComposeFile(b, composeName)
		if err != nil {
			return fmt.Errorf("Could not get compose file: %s - %v", composeFilePath, err)
		}
		composeFilePaths[i] = composeFilePath
	}

	ID := composeNames[0] + "-service"
	if isStack {
		ID = composeNames[0] + "-stack"
	}
	persistedEnv := state.Recover(ID, config.Op.Workspace)

	err := executeCompose(sm, isStack, composeNames, []string{"down"}, persistedEnv)
	if err != nil {
		return fmt.Errorf("Could not stop compose file: %v - %v", composeFilePaths, err)
	}
	defer state.Destroy(ID, config.Op.Workspace)

	log.WithFields(log.Fields{
		"composeFilePath": composeFilePaths,
		"stack":           composeNames[0],
	}).Debug("Docker compose down.")

	return nil
}

func executeCompose(sm *DockerServiceManager, isStack bool, composeNames []string, command []string, env map[string]string) error {
	composeFilePaths := make([]string, len(composeNames))
	for i, composeName := range composeNames {
		b := false
		if i == 0 && isStack {
			b = true
		}

		composeFilePath, err := config.GetComposeFile(b, composeName)
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

	suffix := "-service"
	if isStack {
		suffix = "-stack"
	}
	ID := filepath.Base(filepath.Dir(composeFilePaths[0])) + suffix
	defer state.Update(ID, config.Op.Workspace, composeFilePaths, env)

	log.WithFields(log.Fields{
		"cmd":              command,
		"composeFilePaths": composeFilePaths,
		"env":              env,
		"stack":            composeNames[0],
	}).Debug("Docker compose executed.")

	return nil
}
