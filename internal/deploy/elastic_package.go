// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/config"
	"github.com/elastic/e2e-testing/internal/io"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/testcontainers/testcontainers-go"
	tc "github.com/testcontainers/testcontainers-go"
	"go.elastic.co/apm"
)

const elasticPackagePrefix = "elastic-package-stack"

// elasticPackageBaseCommand represents the command to run 'elastic-package'. As it's declared as a dependency (see tools.go)
// we will run it with 'go run github.com/elastic/elastic-package' instead of running the binary from GOPATH.
var elasticPackageBaseCommand = []string{"run", "github.com/elastic/elastic-package"}

// EPServiceManager manages lifecycle of a service using elastic-package tool
type EPServiceManager struct {
	Context context.Context
}

func newElasticPackage() Deployment {
	return &EPServiceManager{
		Context: context.Background(),
	}
}

// Add adds services deployment: the first service in the list must be the profile in which to deploy the service
func (ep *EPServiceManager) Add(ctx context.Context, profile ServiceRequest, services []ServiceRequest, env map[string]string) error {
	version := common.ElasticAgentVersion

	span, _ := apm.StartSpanOptions(ctx, "Adding elastic-agent to Elastic-Package deployment", "elastic-package.elastic-agent.add", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("elasticAgentVersion", version)
	span.Context.SetLabel("profile", profile)
	span.Context.SetLabel("services", services)
	defer span.End()

	if profile.Name != "fleet" {
		return fmt.Errorf("profile %s not supported in elastic-package provisioner. Services: %v", profile.Name, services)
	}

	for _, srv := range services {
		if srv.Name != "elastic-agent" {
			if srv.Name != common.ElasticAgentServiceName {
				return fmt.Errorf("service %s not supported in elastic-package provisioner. Profile: %s", srv.Name, profile.Name)
			}
		}

		_, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: buildElasticAgentRequest(srv, env),
			Started:          true,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func checkElasticPackageProfile(ctx context.Context, kibanaProfile string) error {
	// check compose profile
	kibanaProfileFile := filepath.Join(config.OpDir(), "compose", "profiles", "fleet", kibanaProfile, "kibana.config.yml")
	found, err := io.Exists(kibanaProfileFile)
	if !found || err != nil {
		return err
	}

	args := append(elasticPackageBaseCommand, "profiles", "create", kibanaProfile, "--from", "default")

	span, _ := apm.StartSpanOptions(ctx, "Copying Elastic Package profile", "elastic-package.profile.create", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("args", args)
	span.Context.SetLabel("kibanaProfile", kibanaProfile)

	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	elasticPackageProfile := filepath.Join(home, ".elastic-package", "profiles", kibanaProfile)
	found, err = io.Exists(elasticPackageProfile)
	if err != nil {
		return err
	}

	if !found {
		_, err = shell.Execute(ctx, ".", "go", args...)
		if err != nil {
			return err
		}
	} else {
		log.Trace("Not creating a new Elastic Package profile for " + kibanaProfile + ". Kibana config will be overriden")
	}

	// The kibana config file is only valid in 8.0.0, for other maintenance branches it's kibana.config.default.yml
	elasticPackageProfileFile := filepath.Join(elasticPackageProfile, "stack", "kibana.config.default.yml")

	// copy compose's kibana's config to elastic-package's config
	err = io.CopyFile(kibanaProfileFile, elasticPackageProfileFile, 10000)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Impossible to copy file")
		return err
	}

	log.WithFields(log.Fields{
		"src":    kibanaProfileFile,
		"target": elasticPackageProfileFile,
	}).Debug("Kibana profile copied")

	return err
}

// Bootstrap sets up environment with docker compose
func (ep *EPServiceManager) Bootstrap(ctx context.Context, profile ServiceRequest, env map[string]string, waitCB func() error) error {
	services := "elasticsearch,fleet-server,kibana"

	version := common.StackVersion

	elasticPackageProfile := "default"
	if kibanaProfile, ok := env["kibanaProfile"]; ok {
		elasticPackageProfile = kibanaProfile
	}

	err := checkElasticPackageProfile(ctx, elasticPackageProfile)
	if err != nil {
		return err
	}

	args := append(elasticPackageBaseCommand, "stack", "up", "--daemon", "--verbose", "--version", version, "--services", services, "-p", elasticPackageProfile)

	span, _ := apm.StartSpanOptions(ctx, "Bootstrapping Elastic Package deployment", "elastic-package.manifest.bootstrap", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("args", args)
	span.Context.SetLabel("profile", profile)
	span.Context.SetLabel("services", services)
	span.Context.SetLabel("stackVersion", version)
	defer span.End()

	if profile.Name != "fleet" {
		return fmt.Errorf("profile %s not supported in elastic-package provisioner. Services: %v", profile.Name, services)
	}

	_, err = shell.ExecuteWithEnv(ctx, ".", "go", env, args...)
	return err
}

// AddFiles - add files to service
func (ep *EPServiceManager) AddFiles(ctx context.Context, profile ServiceRequest, service ServiceRequest, files []string) error {
	// TODO: profile is not used because we are using the docker client, not docker-compose, to reach the service
	span, _ := apm.StartSpanOptions(ctx, "Adding files to Elastic-Package deployment", "elastic-package.files.add", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("files", files)
	span.Context.SetLabel("profile", profile)
	span.Context.SetLabel("service", service)
	defer span.End()

	manifest, _ := ep.GetServiceManifest(ctx, service)
	for _, file := range files {
		isTar := true
		fileExt := filepath.Ext(file)
		if fileExt == ".rpm" || fileExt == ".deb" {
			isTar = false
		}
		err := CopyFileToContainer(ctx, manifest.Name, file, "/", isTar)
		if err != nil {
			log.WithField("error", err).Fatal("Unable to copy file to service")
		}
	}
	return nil
}

// Destroy teardown docker environment
func (ep *EPServiceManager) Destroy(ctx context.Context, profile ServiceRequest) error {
	span, _ := apm.StartSpanOptions(ctx, "Destroying Elastic-Package deployment", "elastic-package.manifest.destroy", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("profile", profile)
	defer span.End()

	if profile.Name != "fleet" {
		return fmt.Errorf("profile %s not supported in elastic-package provisioner", profile.Name)
	}

	_, err := shell.Execute(ctx, ".", "go", append(elasticPackageBaseCommand, "stack", "down", "--verbose")...)
	return err
}

// ExecIn execute command in service
func (ep *EPServiceManager) ExecIn(ctx context.Context, profile ServiceRequest, service ServiceRequest, cmd []string) (string, error) {
	span, _ := apm.StartSpanOptions(ctx, "Executing command in Elastic-Package deployment", "elastic-package.manifest.execIn", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("profile", profile)
	span.Context.SetLabel("service", service)
	span.Context.SetLabel("arguments", cmd)
	defer span.End()

	if profile.Name != "fleet" {
		return "", fmt.Errorf("profile %s not supported in elastic-package provisioner. Service: %v", profile.Name, service)
	}

	manifest, _ := ep.GetServiceManifest(ctx, service)

	args := []string{"exec", "-u", "root", "-i", manifest.Name}
	args = append(args, cmd...)

	output, err := shell.Execute(ctx, ".", "docker", args...)
	if err != nil {
		return "", err
	}
	return output, nil
}

// GetServiceManifest inspects a service
func (ep *EPServiceManager) GetServiceManifest(ctx context.Context, service ServiceRequest) (*ServiceManifest, error) {
	span, _ := apm.StartSpanOptions(ctx, "Inspecting Elastic Package deployment", "elastic-package.manifest.inspect", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("service", service)
	defer span.End()

	inspect, err := InspectContainer(service)
	if err != nil {
		return &ServiceManifest{}, err
	}

	sm := &ServiceManifest{
		ID:         inspect.ID,
		Name:       strings.TrimPrefix(inspect.Name, "/"),
		Connection: service.Name,
		Alias:      inspect.NetworkSettings.Networks["elastic-package-stack_default"].Aliases[0],
		Hostname:   inspect.Config.Hostname,
		Platform:   inspect.Platform,
	}

	log.WithFields(log.Fields{
		"alias":      sm.Alias,
		"connection": sm.Connection,
		"hostname":   sm.Hostname,
		"ID":         sm.ID,
		"name":       sm.Name,
		"platform":   sm.Platform,
	}).Trace("Service Manifest found")

	return sm, nil
}

// Logs print logs of service
func (ep *EPServiceManager) Logs(ctx context.Context, service ServiceRequest) error {
	span, _ := apm.StartSpanOptions(ctx, "Retrieving Elastic Package logs", "elastic-package.manifest.logs", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("service", service)
	defer span.End()

	manifest, _ := ep.GetServiceManifest(context.Background(), service)
	logs, err := shell.Execute(ep.Context, ".", "docker", "logs", manifest.Name)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"service": service.Name,
		}).Error("Could not retrieve Elastic Agent logs")

		return err
	}
	// print logs as is, including tabs and line breaks
	fmt.Println(logs)
	return nil
}

// PreBootstrap sets up environment with the elastic-package tool
func (ep *EPServiceManager) PreBootstrap(ctx context.Context) error {
	span, _ := apm.StartSpanOptions(ctx, "Pre-bootstrapping elastic-package deployment", "elastic-package.bootstrap.pre", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	return nil
}

// Remove remove services from deployment
func (ep *EPServiceManager) Remove(ctx context.Context, profile ServiceRequest, services []ServiceRequest, env map[string]string) error {
	span, _ := apm.StartSpanOptions(ctx, "Removing services from Elastic Package deployment", "elastic-package.services.remove", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("profile", profile)
	span.Context.SetLabel("services", services)
	defer span.End()

	if profile.Name != "fleet" {
		return fmt.Errorf("profile %s not supported in elastic-package provisioner. Services: %v", profile.Name, services)
	}

	for _, service := range services {
		manifest, inspectErr := ep.GetServiceManifest(context.Background(), service)
		if inspectErr != nil {
			log.Warnf("Service %s could not be deleted: %v", service.Name, inspectErr)
			continue
		}

		_, err := shell.Execute(ep.Context, ".", "docker", "rm", "-fv", manifest.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

// Start a container
func (ep *EPServiceManager) Start(ctx context.Context, service ServiceRequest) error {
	span, _ := apm.StartSpanOptions(ctx, "Starting service from Elastic Package deployment", "elastic-package.service.start", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("service", service)
	defer span.End()

	manifest, _ := ep.GetServiceManifest(context.Background(), service)
	_, err := shell.Execute(ep.Context, ".", "docker", "start", manifest.Name)
	return err
}

// Stop a container
func (ep *EPServiceManager) Stop(ctx context.Context, service ServiceRequest) error {
	span, _ := apm.StartSpanOptions(ctx, "Stopping service from Elastic Package deployment", "elastic-package.service.stop", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("service", service)
	defer span.End()

	manifest, _ := ep.GetServiceManifest(context.Background(), service)
	_, err := shell.Execute(ep.Context, ".", "docker", "stop", manifest.Name)
	return err
}

func buildElasticAgentRequest(srv ServiceRequest, env map[string]string) tc.ContainerRequest {
	privileged := false
	var containerMounts []tc.ContainerMount
	var entrypoint []string
	imageNamespace := fmt.Sprintf("elastic-agent%s", env["elasticAgentDockerImageSuffix"])
	img := fmt.Sprintf("docker.elastic.co/%s/%s:%s", env["elasticAgentDockerNamespace"], imageNamespace, env["elasticAgentTag"])

	if srv.Flavour == "centos" || srv.Flavour == "debian" {
		imageNamespace = fmt.Sprintf("elastic-agent-%s", srv.Flavour)
		// use observability's systemd base images to install the elastic-agent on them
		img = "docker.elastic.co/observability-ci/" + srv.Flavour + "-systemd:latest"
		entrypoint = []string{"/sbin/init"}
		containerMounts = []tc.ContainerMount{
			{
				Source: tc.GenericBindMountSource{
					HostPath: "/sys/fs/cgroup",
				},
				Target: "/sys/fs/cgroup",
			},
		}
		privileged = true
	} else if srv.Flavour == "cloud" {
		containerMounts = []tc.ContainerMount{
			{Source: tc.GenericVolumeMountSource{Name: "apmVolume"}, Target: "/apm-legacy"},
		}
		env["FLEET_SERVER_ENABLE"] = "1"
		env["FLEET_SERVER_INSECURE_HTTP"] = "1"
		env["ELASTIC_AGENT_CLOUD"] = "1"
		env["APM_SERVER_PATH"] = "/apm-legacy/apm-server/"
		env["STATE_PATH"] = "/apm-legacy/elastic-agent/"
		env["DATA_PATH"] = "/apm-legacy/data/"
		env["LOGS_PATH"] = "/apm-legacy/logs/"
		env["HOME_PATH"] = "/apm-legacy/"
	} else if srv.Flavour == "" {
		// Docker image for the agent
		env["FLEET_SERVER_ENABLE"] = env["fleetServerMode"]
		env["FLEET_SERVER_INSECURE_HTTP"] = env["fleetServerMode"]
		env["FLEET_ENROLL"] = env["fleetEnroll"]
		env["FLEET_ENROLLMENT_TOKEN"] = env["fleetEnrollmentToken"]
		env["FLEET_INSECURE"] = env["fleetInsecure"]
		env["FLEET_URL"] = env["fleetUrl"]
	}

	req := tc.ContainerRequest{
		Mounts: containerMounts,
		Env:    env,
		Image:  img,
		Labels: map[string]string{
			"name":                       srv.Name, //label is important to handle Inspect,
			"com.docker.compose.project": "elastic-package-stack",
		},
		Name:       fmt.Sprintf("%s_%s_%s_%d", elasticPackagePrefix, imageNamespace, uuid.New().String(), srv.Scale),
		Privileged: privileged,
		Networks:   []string{elasticPackagePrefix + "_default"},
		SkipReaper: common.DeveloperMode, // skip reaping the container if developers required it using the "DEVELOPER_MODE=true" env var
	}

	if len(entrypoint) > 0 {
		req.Entrypoint = entrypoint
	}

	return req
}
