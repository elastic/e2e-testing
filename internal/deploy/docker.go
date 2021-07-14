// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/elastic/e2e-testing/internal/shell"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// DockerDeploymentManifest deploy manifest for docker
type dockerDeploymentManifest struct {
	Context          context.Context
	ConnectionString string
}

func newDockerDeploy() Deployment {
	connectionString := shell.GetEnv("DOCKER_HOST", "")
	return &dockerDeploymentManifest{
		Context:          context.Background(),
		ConnectionString: connectionString,
	}
}

// Add adds services deployment: the first service in the list must be the profile in which to deploy the service
func (c *dockerDeploymentManifest) Add(ctx context.Context, profile ServiceRequest, services []ServiceRequest, env map[string]string) error {
	span, _ := apm.StartSpanOptions(ctx, "Adding services to Docker Compose deployment", "docker-compose.manifest.add-services", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("profile", profile)
	span.Context.SetLabel("services", services)
	defer span.End()

	serviceManager := NewServiceManager()

	return serviceManager.AddServicesToCompose(c.Context, profile, services, env)
}

// Bootstrap sets up environment with docker compose
func (c *dockerDeploymentManifest) Bootstrap(ctx context.Context, profile ServiceRequest, env map[string]string, waitCB func() error) error {
	span, _ := apm.StartSpanOptions(ctx, "Bootstrapping Docker Compose deployment", "docker-compose.manifest.bootstrap", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	serviceManager := NewServiceManager()

	err := serviceManager.RunCompose(ctx, profile, []ServiceRequest{}, env)
	if err != nil {
		log.WithFields(log.Fields{
			"profile": profile,
			"error":   err.Error(),
		}).Fatal("Could not run the runtime dependencies for the profile.")
	}
	err = waitCB()
	if err != nil {
		return err
	}
	return nil
}

// AddFiles - add files to service
func (c *dockerDeploymentManifest) AddFiles(ctx context.Context, profile ServiceRequest, service ServiceRequest, files []string) error {
	// TODO: profile is not used because we are using the docker client, not docker-compose, to reach the service
	span, _ := apm.StartSpanOptions(ctx, "Adding files to Docker Compose deployment", "docker-compose.files.add", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("files", files)
	span.Context.SetLabel("profile", profile)
	span.Context.SetLabel("service", service)
	defer span.End()

	container, _ := c.Inspect(ctx, service)
	for _, file := range files {
		isTar := true
		fileExt := filepath.Ext(file)
		if fileExt == ".rpm" || fileExt == ".deb" {
			isTar = false
		}
		err := CopyFileToContainer(c.Context, container.Name, file, "/", isTar)
		if err != nil {
			log.WithField("error", err).Fatal("Unable to copy file to service")
		}
	}
	return nil
}

// Destroy teardown docker environment
func (c *dockerDeploymentManifest) Destroy(ctx context.Context, profile ServiceRequest) error {
	span, _ := apm.StartSpanOptions(ctx, "Destroying compose deployment", "docker-compose.manifest.destroy", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("profile", profile)
	defer span.End()

	serviceManager := NewServiceManager()
	err := serviceManager.StopCompose(ctx, profile)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"profile": profile,
		}).Fatal("Could not destroy the runtime dependencies for the profile.")
	}
	return nil
}

// ExecIn execute command in service
func (c *dockerDeploymentManifest) ExecIn(ctx context.Context, profile ServiceRequest, service ServiceRequest, cmd []string) (string, error) {
	// TODO: profile is not used because we are using the docker client, not docker-compose, to reach the service
	span, _ := apm.StartSpanOptions(ctx, "Executing command in compose deployment", "docker-compose.manifest.execIn", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("profile", profile)
	span.Context.SetLabel("service", service)
	span.Context.SetLabel("arguments", cmd)
	defer span.End()

	inspect, _ := c.Inspect(ctx, service)
	args := []string{"exec", "-u", "root", "-i", inspect.Name}
	for _, cmdArg := range cmd {
		args = append(args, cmdArg)
	}
	output, err := shell.Execute(ctx, ".", "docker", args...)
	if err != nil {
		return "", err
	}
	return output, nil
}

// Inspect inspects a service
func (c *dockerDeploymentManifest) Inspect(ctx context.Context, service ServiceRequest) (*ServiceManifest, error) {
	span, _ := apm.StartSpanOptions(ctx, "Inspecting compose deployment", "docker-compose.manifest.inspect", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("service", service)
	defer span.End()

	inspect, err := InspectContainer(service)
	if err != nil {
		return &ServiceManifest{}, err
	}

	return &ServiceManifest{
		ID:         inspect.ID,
		Name:       strings.TrimPrefix(inspect.Name, "/"),
		Connection: service.Name,
		Alias:      inspect.NetworkSettings.Networks["fleet_default"].Aliases[0],
		Hostname:   inspect.Config.Hostname,
		Platform:   inspect.Platform,
	}, nil
}

// Logs print logs of service
func (c *dockerDeploymentManifest) Logs(service ServiceRequest) error {
	manifest, _ := c.Inspect(context.Background(), service)
	_, err := shell.Execute(c.Context, ".", "docker", "logs", manifest.Name)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"service": service.Name,
		}).Error("Could not retrieve Elastic Agent logs")

		return err
	}
	return nil
}

// PreBootstrap sets up environment with docker compose
func (c *dockerDeploymentManifest) PreBootstrap(ctx context.Context) error {
	// Check for a docker connection string, this could be a remote docker
	// instance accessible via ssh.
	if c.ConnectionString != "" {
		remoteContextName := "e2e-remote"
		// clear out existing remote context
		args := []string{"context", "rm", remoteContextName}
		_, err := shell.Execute(ctx, ".", "docker", args...)
		if err != nil {
			log.WithField("error", err).Warn("Could not find remote context to remove")
		}

		log.WithField("connectionString", c.ConnectionString).Trace("Connecting to remote docker")
		args = []string{"context", "create", remoteContextName, "--docker", fmt.Sprintf("host=%s", c.ConnectionString)}
		_, err = shell.Execute(ctx, ".", "docker", args...)
		if err != nil {
			return err
		}

		// Switch to the context
		args = []string{"context", "use", remoteContextName}
		_, err = shell.Execute(ctx, ".", "docker", args...)
		if err != nil {
			return err
		}
	} else {
		log.Trace("Using docker default context")
		// Always make sure we use the default docker context if no connection ConnectionString
		// is defined
		args := []string{"context", "use", "default"}
		_, err := shell.Execute(ctx, ".", "docker", args...)
		if err != nil {
			return err
		}
	}

	return nil
}

// Remove remove services from deployment
func (c *dockerDeploymentManifest) Remove(profile ServiceRequest, services []ServiceRequest, env map[string]string) error {
	// TODO: profile is not used because we are using the docker client, not docker-compose, to reach the service
	for _, service := range services {
		manifest, _ := c.Inspect(context.Background(), service)
		_, err := shell.Execute(c.Context, ".", "docker", "rm", "-fv", manifest.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

// Start a container
func (c *dockerDeploymentManifest) Start(service ServiceRequest) error {
	manifest, _ := c.Inspect(context.Background(), service)
	_, err := shell.Execute(c.Context, ".", "docker", "start", manifest.Name)
	return err
}

// Stop a container
func (c *dockerDeploymentManifest) Stop(service ServiceRequest) error {
	manifest, _ := c.Inspect(context.Background(), service)
	_, err := shell.Execute(c.Context, ".", "docker", "stop", manifest.Name)
	return err
}
