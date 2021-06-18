// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ServiceRequest_GetName(t *testing.T) {
	t.Run("ServiceRequest without flavour", func(t *testing.T) {
		srv := NewServiceRequest("foo")

		assert.Equal(t, "foo", srv.GetName(), "Service name has no flavour")
		assert.Equal(t, "foo", srv.GetRealFlavour(), "Flavour matches with service name")
	})

	t.Run("ServiceRequest including flavour", func(t *testing.T) {
		srv := NewServiceRequest("foo").WithFlavour("bar")

		assert.Equal(t, filepath.Join("foo", "bar"), srv.GetName(), "Service name includes flavour")
		assert.Equal(t, "bar", srv.GetRealFlavour(), "Flavour matches with latest flavour")
	})

	t.Run("ServiceRequest including flavour with subdirs", func(t *testing.T) {
		srv := NewServiceRequest("foo").WithFlavour("bar-baaz")

		assert.Equal(t, filepath.Join("foo", "bar", "baaz"), srv.GetName(), "Service name includes flavour with subdir")
		assert.Equal(t, "baaz", srv.GetRealFlavour(), "Flavour matches with latest flavour")
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
