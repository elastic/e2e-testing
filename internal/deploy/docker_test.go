// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"testing"

	"github.com/elastic/e2e-testing/pkg/downloads"
	"github.com/stretchr/testify/assert"
)

func TestGetDockerNamespaceEnvVar(t *testing.T) {
	t.Run("Returns fallback when environment variable is not set", func(t *testing.T) {
		namespace := GetDockerNamespaceEnvVar("beats")
		assert.True(t, namespace == "beats")
	})

	t.Run("Returns Observability CI when environment variable is set", func(t *testing.T) {
		downloads.GithubCommitSha1 = "0123456789"
		defer func() { downloads.GithubCommitSha1 = "" }()

		namespace := GetDockerNamespaceEnvVar("beats")
		assert.True(t, namespace == "observability-ci")
	})
}

func TestGetDockerNamespaceEnvVarForRepository(t *testing.T) {
	t.Run("Returns fallback when environment variables are not set", func(t *testing.T) {
		namespace := GetDockerNamespaceEnvVarForRepository("filebeat", "beats")
		assert.True(t, namespace == "beats")
	})

	t.Run("Returns Observability CI when environment variable is set for Beats", func(t *testing.T) {
		downloads.GithubCommitSha1 = "0123456789"
		downloads.GithubRepository = "beats"
		defer func() {
			downloads.GithubCommitSha1 = ""
			downloads.GithubRepository = "elastic-agent"
		}()

		namespace := GetDockerNamespaceEnvVarForRepository("filebeat", "beats")
		assert.True(t, namespace == "observability-ci")
	})

	t.Run("Returns Observability CI when environment variable is set for elastic-agent", func(t *testing.T) {
		downloads.GithubCommitSha1 = "0123456789"
		defer func() {
			downloads.GithubCommitSha1 = ""
			downloads.GithubRepository = "elastic-agent"
		}()

		namespace := GetDockerNamespaceEnvVarForRepository("elastic-agent", "beats")
		assert.True(t, namespace == "observability-ci")
	})
}
