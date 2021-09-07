// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDockerNamespaceEnvVar(t *testing.T) {
	t.Run("Returns fallback when environment variable is not set", func(t *testing.T) {
		namespace := GetDockerNamespaceEnvVar("beats")
		assert.True(t, namespace == "beats")
	})

	t.Run("Returns Observability CI when environment variable is set", func(t *testing.T) {
		defer os.Unsetenv("BEATS_USE_CI_SNAPSHOTS")
		os.Setenv("BEATS_USE_CI_SNAPSHOTS", "true")

		namespace := GetDockerNamespaceEnvVar("beats")
		assert.True(t, namespace == "observability-ci")
	})
}
