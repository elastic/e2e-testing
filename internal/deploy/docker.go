// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// DockerDeploymentManifest deploy manifest for docker
type dockerDeploymentManifest struct {
	Context context.Context
}

func newDockerDeploy() Deployment {
	return &dockerDeploymentManifest{Context: context.Background()}
}

// Add adds services deployment
func (c *dockerDeploymentManifest) Add(services []ServiceRequest, env map[string]string) error {
	serviceManager := NewServiceManager()

	return serviceManager.AddServicesToCompose(c.Context, services[0], services[1:], env)
}

// Bootstrap sets up environment with docker compose
func (c *dockerDeploymentManifest) Bootstrap(waitCB func() error) error {
	serviceManager := NewServiceManager()
	common.ProfileEnv = map[string]string{
		"kibanaVersion": common.KibanaVersion,
		"stackPlatform": "linux/" + utils.GetArchitecture(),
		"stackVersion":  common.StackVersion,
	}

	common.ProfileEnv["kibanaDockerNamespace"] = "kibana"
	if strings.HasPrefix(common.KibanaVersion, "pr") || utils.IsCommit(common.KibanaVersion) {
		// because it comes from a PR
		common.ProfileEnv["kibanaDockerNamespace"] = "observability-ci"
	}

	profile := NewServiceRequest(common.FleetProfileName)
	err := serviceManager.RunCompose(c.Context, true, []ServiceRequest{profile}, common.ProfileEnv)
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
func (c *dockerDeploymentManifest) AddFiles(service ServiceRequest, files []string) error {
	container, _ := c.Inspect(service)
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
func (c *dockerDeploymentManifest) Destroy(ctx context.Context) error {
	span, _ := apm.StartSpanOptions(ctx, "Destroying compose deployment", "docker-compose.manifest.destroy", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	serviceManager := NewServiceManager()
	err := serviceManager.StopCompose(ctx, true, []ServiceRequest{NewServiceRequest(common.FleetProfileName)})
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"profile": common.FleetProfileName,
		}).Fatal("Could not destroy the runtime dependencies for the profile.")
	}
	return nil
}

// ExecIn execute command in service
func (c *dockerDeploymentManifest) ExecIn(service ServiceRequest, cmd []string) (string, error) {
	inspect, _ := c.Inspect(service)
	args := []string{"exec", "-u", "root", "-i", inspect.Name}
	for _, cmdArg := range cmd {
		args = append(args, cmdArg)
	}
	output, err := shell.Execute(c.Context, ".", "docker", args...)
	if err != nil {
		return "", err
	}
	return output, nil
}

// Inspect inspects a service
func (c *dockerDeploymentManifest) Inspect(service ServiceRequest) (*ServiceManifest, error) {
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
	manifest, _ := c.Inspect(service)
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

// Remove remove services from deployment
func (c *dockerDeploymentManifest) Remove(services []ServiceRequest, env map[string]string) error {
	for _, service := range services[1:] {
		manifest, _ := c.Inspect(service)
		_, err := shell.Execute(c.Context, ".", "docker", "rm", "-fv", manifest.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

// Start a container
func (c *dockerDeploymentManifest) Start(service ServiceRequest) error {
	manifest, _ := c.Inspect(service)
	_, err := shell.Execute(c.Context, ".", "docker", "start", manifest.Name)
	return err
}

// Stop a container
func (c *dockerDeploymentManifest) Stop(service ServiceRequest) error {
	manifest, _ := c.Inspect(service)
	_, err := shell.Execute(c.Context, ".", "docker", "stop", manifest.Name)
	return err
}
