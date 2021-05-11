// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"strings"
)

// Deployment interface for operations dealing with deployments of the bits
// required for testing
type Deployment interface {
	Add(services []ServiceRequest, env map[string]string) error // adds a service to deployment
	Bootstrap(waitCB func() error) error                        // will bootstrap or reuse existing cluster if kubernetes is selected
	Destroy() error                                             // Teardown deployment
	ExecIn(service string, cmd []string) (string, error)        // Execute arbitrary commands in service
	Inspect(service string) (*ServiceManifest, error)           // inspects service
	Remove(services []string, env map[string]string) error      // Removes services from deployment
}

// ServiceManifest information about a service in a deployment
type ServiceManifest struct {
	ID         string
	Name       string
	Connection string // a string representing how to connect to service
	Hostname   string
}

// ServiceRequest represents the service to be created using the provider
type ServiceRequest struct {
	Name    string
	Flavour string // optional, configured using builder method
}

// NewServiceRequest creates a request for a service
func NewServiceRequest(n string) ServiceRequest {
	return ServiceRequest{
		Name: n,
	}
}

// WithFlavour adds a flavour for the service, resulting in a look-up of the service in the config directory,
// using flavour as a subdir of the service
func (sr ServiceRequest) WithFlavour(f string) ServiceRequest {
	sr.Flavour = f
	return sr
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
