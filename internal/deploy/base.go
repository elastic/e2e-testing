// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"strings"

	"github.com/elastic/e2e-testing/internal/installer"
)

// Deployment interface for operations dealing with deployments of the bits
// required for testing
type Deployment interface {
	Add(services []string, env map[string]string) error                  // adds a service to deployment
	AddFiles(service string, files []string) error                       // adds files to a service
	Bootstrap(waitCB func() error) error                                 // will bootstrap or reuse existing cluster if kubernetes is selected
	Destroy() error                                                      // Teardown deployment
	ExecIn(service string, cmd []string) (string, error)                 // Execute arbitrary commands in service
	Inspect(service string) (*ServiceManifest, error)                    // inspects service
	Mount(service string, installType string) (installer.Package, error) // mounts a service for performing actions against it
	Remove(services []string, env map[string]string) error               // Removes services from deployment
	Start(service string) error                                          // Starts a service or container depending on Deployment
	Stop(service string) error                                           // Stop a service or container depending on deployment
}

// ServiceManifest information about a service in a deployment
type ServiceManifest struct {
	ID         string
	Name       string
	Connection string // a string representing how to connect to service
	Alias      string // docker network aliases
	Hostname   string
	Platform   string // running in linux, macos, windows
}

// New creates a new deployment
func New(provider string) Deployment {
	if strings.EqualFold(provider, "docker") {
		return newDockerDeploy()
	}
	if strings.EqualFold(provider, "kubernetes") {
		return newK8sDeploy()
	}
	return nil
}
