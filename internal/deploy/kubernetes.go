// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/e2e-testing/internal/kubernetes"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

var cluster kubernetes.Cluster
var kubectl kubernetes.Control

// KubernetesDeploymentManifest deploy manifest for kubernetes
type kubernetesDeploymentManifest struct {
	Context context.Context
}

func newK8sDeploy() Deployment {
	return &kubernetesDeploymentManifest{Context: context.Background()}
}

// Add adds services deployment
func (c *kubernetesDeploymentManifest) Add(ctx context.Context, profile ServiceRequest, services []ServiceRequest, env map[string]string) error {
	span, _ := apm.StartSpanOptions(ctx, "Adding services to kubernetes deployment", "kubernetes.manifest.add-services", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("profile", profile)
	span.Context.SetLabel("services", services)
	defer span.End()

	kubectl = cluster.Kubectl().WithNamespace(ctx, getNamespaceFromProfile(profile))

	for _, service := range services {
		_, err := kubectl.Run(ctx, "apply", "-k", fmt.Sprintf("../../../internal/config/kubernetes/overlays/%s", service.Name))
		if err != nil {
			return err
		}
	}
	return nil
}

// AddFiles - add files to deployment service
func (c *kubernetesDeploymentManifest) AddFiles(ctx context.Context, profile ServiceRequest, service ServiceRequest, files []string) error {
	span, _ := apm.StartSpanOptions(ctx, "Adding files to kubernetes deployment", "kubernetes.files.add", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("files", files)
	span.Context.SetLabel("profile", profile)
	span.Context.SetLabel("service", service)
	defer span.End()

	manifest, _ := c.GetServiceManifest(ctx, service)
	kubectl = cluster.Kubectl().WithNamespace(ctx, getNamespaceFromProfile(profile))

	for _, file := range files {
		_, err := kubectl.Run(ctx, "cp", file, fmt.Sprintf("deployment/%s:.", manifest.Name))
		if err != nil {
			log.WithField("error", err).Fatal("Unable to copy file to service")
		}
	}
	return nil
}

// Bootstrap sets up environment with kind
func (c *kubernetesDeploymentManifest) Bootstrap(ctx context.Context, profile ServiceRequest, env map[string]string, waitCB func() error) error {
	span, _ := apm.StartSpanOptions(ctx, "Bootstrapping kubernetes deployment", "kubernetes.manifest.bootstrap", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	err := cluster.Initialize(ctx, "../../../internal/config/kubernetes/kind.yaml")
	if err != nil {
		return err
	}

	// TODO: we would need to understand how to pass the environment argument to anything running in the namespace
	kubectl = cluster.Kubectl().WithNamespace(ctx, getNamespaceFromProfile(profile))
	_, err = kubectl.Run(ctx, "apply", "-k", "../../../internal/config/kubernetes/base")
	if err != nil {
		return err
	}
	err = waitCB()
	if err != nil {
		return err
	}
	return nil
}

// Destroy teardown kubernetes environment
func (c *kubernetesDeploymentManifest) Destroy(ctx context.Context, profile ServiceRequest) error {
	span, _ := apm.StartSpanOptions(ctx, "Destroying kubernetes deployment", "kubernetes.manifest.destroy", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	kubectl = cluster.Kubectl().WithNamespace(ctx, getNamespaceFromProfile(profile))
	cluster.Cleanup(c.Context)
	return nil
}

// ExecIn execute command in service
func (c *kubernetesDeploymentManifest) ExecIn(ctx context.Context, profile ServiceRequest, service ServiceRequest, cmd []string) (string, error) {
	span, _ := apm.StartSpanOptions(ctx, "Executing command in kubernetes deployment", "kubernetes.manifest.execIn", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("profile", profile)
	span.Context.SetLabel("service", service)
	span.Context.SetLabel("arguments", cmd)
	defer span.End()

	kubectl = cluster.Kubectl().WithNamespace(ctx, getNamespaceFromProfile(profile))
	args := []string{"exec", "deployment/" + service.Name, "--"}
	for _, arg := range cmd {
		args = append(cmd, arg)
	}
	output, err := kubectl.Run(ctx, args...)
	if err != nil {
		return "", err
	}
	return output, nil
}

type kubernetesServiceManifest struct {
	Metadata struct {
		Name string `json:"name"`
		ID   string `json:"uid"`
	} `json:"metadata"`
}

// GetServiceManifest inspects a service
func (c *kubernetesDeploymentManifest) GetServiceManifest(ctx context.Context, service ServiceRequest) (*ServiceManifest, error) {
	span, _ := apm.StartSpanOptions(ctx, "Inspecting kubernetes deployment", "kubernetes.manifest.inspect", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("service", service)
	defer span.End()

	kubectl = cluster.Kubectl().WithNamespace(ctx, "default")
	out, err := kubectl.Run(ctx, "get", "deployment/"+service.Name, "-o", "json")
	if err != nil {
		return &ServiceManifest{}, err
	}
	var inspect kubernetesServiceManifest
	if err = json.Unmarshal([]byte(out), &inspect); err != nil {
		return &ServiceManifest{}, errors.Wrap(err, "Could not convert metadata to JSON")
	}

	sm := &ServiceManifest{
		ID:         inspect.Metadata.ID,
		Name:       strings.TrimPrefix(inspect.Metadata.Name, "/"),
		Connection: service.Name,
		Hostname:   service.Name,
		Alias:      service.Name,
		Platform:   "linux",
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
func (c *kubernetesDeploymentManifest) Logs(ctx context.Context, service ServiceRequest) error {
	span, _ := apm.StartSpanOptions(ctx, "Retrieving kubernetes logs", "kubernetes.manifest.logs", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("service", service)
	defer span.End()

	kubectl = cluster.Kubectl().WithNamespace(ctx, "default")
	logs, err := kubectl.Run(ctx, "logs", "deployment/"+service.Name)
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

// PreBootstrap sets up environment with kind
func (c *kubernetesDeploymentManifest) PreBootstrap(ctx context.Context) error {
	return nil
}

// Remove remove services from deployment
func (c *kubernetesDeploymentManifest) Remove(ctx context.Context, profile ServiceRequest, services []ServiceRequest, env map[string]string) error {
	span, _ := apm.StartSpanOptions(ctx, "Removing services from kubernetes deployment", "kubernetes.services.remove", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("profile", profile)
	span.Context.SetLabel("services", services)
	defer span.End()

	kubectl = cluster.Kubectl().WithNamespace(c.Context, getNamespaceFromProfile(profile))

	for _, service := range services {
		_, err := kubectl.Run(c.Context, "delete", "-k", fmt.Sprintf("../../../internal/config/kubernetes/overlays/%s", service.Name))
		if err != nil {
			return err
		}
	}
	return nil
}

// Start a container
func (c *kubernetesDeploymentManifest) Start(ctx context.Context, service ServiceRequest) error {
	return nil
}

// Stop a container
func (c *kubernetesDeploymentManifest) Stop(ctx context.Context, service ServiceRequest) error {
	return nil
}

func getNamespaceFromProfile(profile ServiceRequest) string {
	if profile.Name == "" {
		return "default"
	}

	return profile.GetName()
}
