// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"path/filepath"
	"testing"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/stretchr/testify/assert"
)

func Test_New(t *testing.T) {
	t.Run("New Docker Provider", func(t *testing.T) {
		provider := New("docker")

		_, ok := provider.(*dockerDeploymentManifest)
		assert.True(t, ok, "Provider is not Docker")
	})

	t.Run("New Elastic Package Provider", func(t *testing.T) {
		provider := New("elastic-package")

		_, ok := provider.(*EPServiceManager)
		assert.True(t, ok, "Provider is not Elastic Package")
	})

	t.Run("New K8S Provider", func(t *testing.T) {
		provider := New("kubernetes")

		_, ok := provider.(*kubernetesDeploymentManifest)
		assert.True(t, ok, "Provider is not Kubernetes")
	})

	t.Run("New Remote Provider", func(t *testing.T) {
		provider := New("remote")

		_, ok := provider.(*remoteDeploymentManifest)
		assert.True(t, ok, "Provider is not Remote")
	})

	t.Run("New Not Found Provider", func(t *testing.T) {
		provider := New("asdf")

		assert.Nil(t, provider, "Provider is not Nil")
	})
}

func Test_ServiceRequest(t *testing.T) {
	t.Run("ServiceRequest as service", func(t *testing.T) {
		srv := NewServiceRequest("foo")

		assert.False(t, srv.IsContainer, "Service should not be a container")
	})

	t.Run("ServiceRequest as container", func(t *testing.T) {
		srv := NewServiceContainerRequest("foo")

		assert.True(t, srv.IsContainer, "Service should be a container")
	})
}

func Test_ServiceRequest_GetName(t *testing.T) {
	t.Run("ServiceRequest without flavour", func(t *testing.T) {
		srv := NewServiceRequest("foo")

		assert.Equal(t, "foo", srv.GetName(), "Service name has no flavour")
	})

	t.Run("ServiceRequest including flavour", func(t *testing.T) {
		srv := NewServiceRequest("foo").WithFlavour("bar")

		assert.Equal(t, filepath.Join("foo", "bar"), srv.GetName(), "Service name includes flavour")
	})
}

func Test_ServiceRequest_WithScale(t *testing.T) {
	t.Run("ServiceRequest without scale", func(t *testing.T) {
		srv := NewServiceRequest("foo")

		assert.Equal(t, 1, srv.Scale, "Service scale is 1")
	})

	t.Run("ServiceRequest including zero or negative scale", func(t *testing.T) {
		srv := NewServiceRequest("foo").WithScale(0)
		assert.Equal(t, 1, srv.Scale, "Service scale is 1")

		srv = NewServiceRequest("foo").WithScale(-1)
		assert.Equal(t, 1, srv.Scale, "Service scale is 1")
	})

	t.Run("ServiceRequest including scale", func(t *testing.T) {
		srv := NewServiceRequest("foo").WithScale(6)

		assert.Equal(t, 6, srv.Scale, "Service scale is 6")
	})
}

func Test_ServiceRequest_GetVersion(t *testing.T) {
	originalElasticAgentVersion := common.ElasticAgentVersion

	t.Run("ServiceRequest without version", func(t *testing.T) {
		common.ElasticAgentVersion = "1.2.3"
		defer func() {
			common.ElasticAgentVersion = originalElasticAgentVersion
		}()

		srv := NewServiceRequest("foo")

		assert.Equal(t, "1.2.3", srv.Version, "Service has default version")
	})

	t.Run("ServiceRequest including version", func(t *testing.T) {
		common.ElasticAgentVersion = "1.2.3"
		defer func() {
			common.ElasticAgentVersion = originalElasticAgentVersion
		}()

		srv := NewServiceRequest("foo").WithVersion("4.5.6")

		assert.Equal(t, "4.5.6", srv.Version, "Service has version")
	})
}
