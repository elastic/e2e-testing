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
func (c *kubernetesDeploymentManifest) Add(services []ServiceRequest, env map[string]string) error {
	kubectl = cluster.Kubectl().WithNamespace(c.Context, "default")

	for _, service := range services {
		_, err := kubectl.Run(c.Context, "apply", "-k", fmt.Sprintf("../../../cli/config/kubernetes/overlays/%s", service.Name))
		if err != nil {
			return err
		}
	}
	return nil
}

// AddFiles - add files to deployment service
func (c *kubernetesDeploymentManifest) AddFiles(service ServiceRequest, files []string) error {
	container, _ := c.Inspect(service)
	kubectl = cluster.Kubectl().WithNamespace(c.Context, "default")

	for _, file := range files {
		_, err := kubectl.Run(c.Context, "cp", file, fmt.Sprintf("deployment/%s:.", container.Name))
		if err != nil {
			log.WithField("error", err).Fatal("Unable to copy file to service")
		}
	}
	return nil
}

// Bootstrap sets up environment with kind
func (c *kubernetesDeploymentManifest) Bootstrap(waitCB func() error) error {
	err := cluster.Initialize(c.Context, "../../../cli/config/kubernetes/kind.yaml")
	if err != nil {
		return err
	}

	kubectl = cluster.Kubectl().WithNamespace(c.Context, "default")
	_, err = kubectl.Run(c.Context, "apply", "-k", "../../../cli/config/kubernetes/base")
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
func (c *kubernetesDeploymentManifest) Destroy(ctx context.Context) error {
	span, _ := apm.StartSpanOptions(ctx, "Destroying kubernetes deployment", "kubernetes.manifest.destroy", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	kubectl = cluster.Kubectl().WithNamespace(ctx, "default")
	cluster.Cleanup(c.Context)
	return nil
}

// ExecIn execute command in service
func (c *kubernetesDeploymentManifest) ExecIn(service ServiceRequest, cmd []string) (string, error) {
	kubectl = cluster.Kubectl().WithNamespace(c.Context, "default")
	args := []string{"exec", "deployment/" + service.Name, "--"}
	for _, arg := range cmd {
		args = append(cmd, arg)
	}
	output, err := kubectl.Run(c.Context, args...)
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

// Inspect inspects a service
func (c *kubernetesDeploymentManifest) Inspect(service ServiceRequest) (*ServiceManifest, error) {
	kubectl = cluster.Kubectl().WithNamespace(c.Context, "default")
	out, err := kubectl.Run(c.Context, "get", "deployment/"+service.Name, "-o", "json")
	if err != nil {
		return &ServiceManifest{}, err
	}
	var inspect kubernetesServiceManifest
	if err = json.Unmarshal([]byte(out), &inspect); err != nil {
		return &ServiceManifest{}, errors.Wrap(err, "Could not convert metadata to JSON")
	}
	return &ServiceManifest{
		ID:         inspect.Metadata.ID,
		Name:       strings.TrimPrefix(inspect.Metadata.Name, "/"),
		Connection: service.Name,
		Hostname:   service.Name,
		Alias:      service.Name,
		Platform:   "linux",
	}, nil
}

// Logs print logs of service
func (c *kubernetesDeploymentManifest) Logs(service ServiceRequest) error {
	kubectl = cluster.Kubectl().WithNamespace(c.Context, "default")
	_, err := kubectl.Run(c.Context, "logs", "deployment/"+service.Name)
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
func (c *kubernetesDeploymentManifest) Remove(services []ServiceRequest, env map[string]string) error {
	kubectl = cluster.Kubectl().WithNamespace(c.Context, "default")

	for _, service := range services {
		_, err := kubectl.Run(c.Context, "delete", "-k", fmt.Sprintf("../../../cli/config/kubernetes/overlays/%s", service.Name))
		if err != nil {
			return err
		}
	}
	return nil
}

// Start a container
func (c *kubernetesDeploymentManifest) Start(service ServiceRequest) error {
	return nil
}

// Stop a container
func (c *kubernetesDeploymentManifest) Stop(service ServiceRequest) error {
	return nil
}
