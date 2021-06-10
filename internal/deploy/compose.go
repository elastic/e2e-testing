// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/internal/io"
	state "github.com/elastic/e2e-testing/internal/state"
	"go.elastic.co/apm"

	log "github.com/sirupsen/logrus"
	tc "github.com/testcontainers/testcontainers-go"
)

// ServiceManager manages lifecycle of a service
type ServiceManager interface {
	AddServicesToCompose(ctx context.Context, profile ServiceRequest, services []ServiceRequest, env map[string]string) error
	ExecCommandInService(ctx context.Context, profile ServiceRequest, image ServiceRequest, serviceName string, cmds []string, env map[string]string, detach bool) error
	RemoveServicesFromCompose(ctx context.Context, profile ServiceRequest, services []ServiceRequest, env map[string]string) error
	RunCommand(ctx context.Context, services []ServiceRequest, composeArgs []string, env map[string]string) error
	RunCompose(ctx context.Context, isProfile bool, services []ServiceRequest, env map[string]string) error
	StopCompose(ctx context.Context, isProfile bool, services []ServiceRequest) error
}

// DockerServiceManager implementation of the service manager interface
type DockerServiceManager struct {
}

// NewServiceManager returns a new service manager
func NewServiceManager() ServiceManager {
	return &DockerServiceManager{}
}

// AddServicesToCompose adds services to a running docker compose
func (sm *DockerServiceManager) AddServicesToCompose(ctx context.Context, profile ServiceRequest, services []ServiceRequest, env map[string]string) error {
	span, _ := apm.StartSpanOptions(ctx, "Add services to Docker Compose", "docker-compose.services.add", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	log.WithFields(log.Fields{
		"profile":  profile,
		"services": services,
	}).Trace("Adding services to compose")

	scaleCmds := []string{}
	newServices := []ServiceRequest{profile}
	for _, srv := range services {
		newServices = append(newServices, srv)
		if srv.Scale > 1 {
			scaleCmds = append(scaleCmds, fmt.Sprintf("%s=%d", srv.Name, srv.Scale))
		}
	}

	run := state.Recover(profile.Name+"-profile", config.OpDir())
	persistedEnv := run.Env
	for k, v := range env {
		persistedEnv[k] = v
	}

	cmds := []string{"up", "-d"}
	if len(scaleCmds) > 0 {
		cmds = append(cmds, "--scale")
		cmds = append(cmds, scaleCmds...)
	}

	err := executeCompose(ctx, true, newServices, cmds, persistedEnv)
	if err != nil {
		return err
	}

	return nil
}

// ExecCommandInService executes a command in a service from a profile
func (sm *DockerServiceManager) ExecCommandInService(ctx context.Context, profile ServiceRequest, image ServiceRequest, serviceName string, cmds []string, env map[string]string, detach bool) error {
	services := []ServiceRequest{
		profile, // profile name
		image,   // image for the service
	}
	composeArgs := []string{"exec", "-T"}
	if detach {
		composeArgs = append(composeArgs, "-d")
	}
	composeArgs = append(composeArgs, "--index", fmt.Sprintf("%d", image.Scale))
	composeArgs = append(composeArgs, serviceName)
	composeArgs = append(composeArgs, cmds...)

	err := sm.RunCommand(ctx, services, composeArgs, env)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmds,
			"error":   err,
			"service": serviceName,
		}).Error("Could not execute command in service container")

		return err
	}

	return nil
}

// RemoveServicesFromCompose removes services from a running docker compose
func (sm *DockerServiceManager) RemoveServicesFromCompose(ctx context.Context, profile ServiceRequest, services []ServiceRequest, env map[string]string) error {
	span, _ := apm.StartSpanOptions(ctx, "Remove services from Docker Compose", "docker-compose.services.remove", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("profile", profile)
	span.Context.SetLabel("services", services)
	defer span.End()

	log.WithFields(log.Fields{
		"profile":  profile,
		"services": services,
	}).Trace("Removing services from compose")

	newServices := []ServiceRequest{profile}
	newServices = append(newServices, services...)

	run := state.Recover(profile.Name+"-profile", config.OpDir())
	persistedEnv := run.Env
	for k, v := range env {
		persistedEnv[k] = v
	}

	for _, srv := range services {
		command := []string{"rm", "-fvs"}
		command = append(command, srv.Name)

		err := executeCompose(ctx, true, newServices, command, persistedEnv)
		if err != nil {
			log.WithFields(log.Fields{
				"command": command,
				"service": srv,
				"profile": profile,
			}).Error("Could not remove service from compose")
			return err
		}
		log.WithFields(log.Fields{
			"profile": profile,
			"service": srv,
		}).Debug("Service removed from compose")
	}

	return nil
}

// RunCommand executes a docker-compose command in a running a docker compose
func (sm *DockerServiceManager) RunCommand(ctx context.Context, services []ServiceRequest, composeArgs []string, env map[string]string) error {
	return executeCompose(ctx, true, services, composeArgs, env)
}

// RunCompose runs a docker compose by its name
func (sm *DockerServiceManager) RunCompose(ctx context.Context, isProfile bool, services []ServiceRequest, env map[string]string) error {
	span, _ := apm.StartSpanOptions(ctx, "Starting Docker Compose files", "docker-compose.services.up", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("services", services)
	defer span.End()

	return executeCompose(ctx, isProfile, services, []string{"up", "-d"}, env)
}

// StopCompose stops a docker compose by its name
func (sm *DockerServiceManager) StopCompose(ctx context.Context, isProfile bool, services []ServiceRequest) error {
	span, _ := apm.StartSpanOptions(ctx, "Stopping Docker Compose files", "docker-compose.services.down", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("services", services)
	defer span.End()

	composeFilePaths := make([]string, len(services))
	for i, srv := range services {
		b := isProfile
		if i == 0 && !isProfile && (len(services) == 1) {
			b = true
		}

		serviceIncludingFlavour := srv.Name
		if srv.Flavour != "" {
			// discover the flavour in the subdir
			serviceIncludingFlavour = filepath.Join(srv.Name, srv.Flavour)
		}

		composeFilePath, err := getComposeFile(b, serviceIncludingFlavour)
		if err != nil {
			return fmt.Errorf("Could not get compose file: %s - %v", composeFilePath, err)
		}
		composeFilePaths[i] = composeFilePath
	}

	ID := services[0].Name + "-service"
	if isProfile {
		ID = services[0].Name + "-profile"
	}
	run := state.Recover(ID, config.OpDir())
	persistedEnv := run.Env

	err := executeCompose(ctx, isProfile, services, []string{"down", "--remove-orphans"}, persistedEnv)
	if err != nil {
		return fmt.Errorf("Could not stop compose file: %v - %v", composeFilePaths, err)
	}
	defer state.Destroy(ID, config.OpDir())

	log.WithFields(log.Fields{
		"composeFilePath": composeFilePaths,
		"profile":         services[0].Name,
	}).Trace("Docker compose down.")

	return nil
}

func executeCompose(ctx context.Context, isProfile bool, services []ServiceRequest, command []string, env map[string]string) error {
	span, _ := apm.StartSpanOptions(ctx, "Executing Docker Compose command", "docker-compose.services.exec", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("services", services)
	span.Context.SetLabel("command", command)
	defer span.End()

	composeFilePaths := make([]string, len(services))
	for i, srv := range services {
		b := false
		if i == 0 && isProfile {
			b = true
		}

		serviceIncludingFlavour := srv.Name
		if srv.Flavour != "" {
			// discover the flavour in the subdir
			serviceIncludingFlavour = filepath.Join(srv.Name, srv.Flavour)
		}

		composeFilePath, err := getComposeFile(b, serviceIncludingFlavour)
		if err != nil {
			return fmt.Errorf("Could not get compose file: %s - %v", composeFilePath, err)
		}
		composeFilePaths[i] = composeFilePath
	}

	compose := tc.NewLocalDockerCompose(composeFilePaths, services[0].Name)
	execError := compose.
		WithCommand(command).
		WithEnv(env).
		Invoke()
	err := execError.Error
	if err != nil {
		return fmt.Errorf("Could not run compose file: %v - %v", composeFilePaths, err)
	}

	suffix := "-service"
	if isProfile {
		suffix = "-profile"
	}
	ID := filepath.Base(filepath.Dir(composeFilePaths[0])) + suffix
	defer state.Update(ID, config.OpDir(), composeFilePaths, env)

	log.WithFields(log.Fields{
		"cmd":              command,
		"composeFilePaths": composeFilePaths,
		"env":              env,
		"profile":          services[0].Name,
	}).Debug("Docker compose executed.")

	return nil
}

// getComposeFile returns the path of the compose file, looking up the
// tool's workdir
func getComposeFile(isProfile bool, composeName string) (string, error) {
	composeFileName := "docker-compose.yml"
	serviceType := "services"
	if isProfile {
		serviceType = "profiles"
	}

	composeFilePath := path.Join(config.OpDir(), "compose", serviceType, composeName, composeFileName)
	found, err := io.Exists(composeFilePath)
	if found && err == nil {
		log.WithFields(log.Fields{
			"composeFilePath": composeFilePath,
			"type":            serviceType,
		}).Trace("Compose file found at workdir")

		return composeFilePath, nil
	}

	log.WithFields(log.Fields{
		"composeFilePath": composeFilePath,
		"error":           err,
		"type":            serviceType,
	}).Trace("Compose file not found. Please make sure the file exists at the location")

	if err != nil {
		log.WithFields(log.Fields{
			"composeFileName": composeFileName,
			"error":           err,
			"isProfile":       isProfile,
			"type":            serviceType,
		}).Error("Could not find compose file.")

		return "", err
	}

	return composeFilePath, nil
}
