// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"context"
	"strings"
)

// Deployment interface for operations dealing with deployments of the bits
// required for testing
type Deployment interface {
	Add(services []ServiceRequest, env map[string]string) error    // adds a service to deployment
	AddFiles(service ServiceRequest, files []string) error         // adds files to a service
	Bootstrap(ctx context.Context, waitCB func() error) error      // will bootstrap or reuse existing cluster if kubernetes is selected
	Destroy(ctx context.Context) error                             // Teardown deployment
	ExecIn(service ServiceRequest, cmd []string) (string, error)   // Execute arbitrary commands in service
	Inspect(service ServiceRequest) (*ServiceManifest, error)      // inspects service
	Logs(service ServiceRequest) error                             // prints logs of deployed service
	Remove(services []ServiceRequest, env map[string]string) error // Removes services from deployment
	Start(service ServiceRequest) error                            // Starts a service or container depending on Deployment
	Stop(service ServiceRequest) error                             // Stop a service or container depending on deployment
}

// ServiceOperator represents the operations that can be performed by a service
type ServiceOperator interface {
	AddFiles(files []string) error             // adds files to service environment
	Enroll(token string) error                 // handle any enrollment/registering of service
	Exec(args []string) (string, error)        // exec arbitrary commands in service environment
	Inspect() (ServiceOperatorManifest, error) // returns manifest for package
	Install() error
	InstallCerts() error
	Logs() error
	Postinstall() error
	Preinstall() error
	Start() error // will start a service
	Stop() error  // will stop a service
	Uninstall() error
}

// ServiceOperatorManifest is state information for each service operator
type ServiceOperatorManifest struct {
	CommitFile string
	WorkDir    string
}

// ServiceManifest information about a service in a deployment
type ServiceManifest struct {
	ID         string
	Name       string
	Connection string // a string representing how to connect to service
	Alias      string // container network aliases
	Hostname   string
	Platform   string // running in linux, macos, windows
}

// ServiceRequest represents the service to be created using the provider
type ServiceRequest struct {
	Name    string
	Flavour string // optional, configured using builder method
	Scale   int    // default: 1
}

// NewServiceRequest creates a request for a service
func NewServiceRequest(n string) ServiceRequest {
	return ServiceRequest{
		Name:  n,
		Scale: 1,
	}
}

// WithFlavour adds a flavour for the service, resulting in a look-up of the service in the config directory,
// using flavour as a subdir of the service
func (sr ServiceRequest) WithFlavour(f string) ServiceRequest {
	sr.Flavour = f
	return sr
}

// WithScale adds the scale index to the service
func (sr ServiceRequest) WithScale(s int) ServiceRequest {
	if s < 1 {
		s = 1
	}

	sr.Scale = s
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
