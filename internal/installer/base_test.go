// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadAgentBinary(t *testing.T) {
	artifact := "elastic-agent"
	beatsDir := path.Join("..", "_testresources", "beats")
	distributionsDir, _ := filepath.Abs(path.Join(beatsDir, "x-pack", "elastic-agent", "build", "distributions"))
	version := "8.0.0-SNAPSHOT"

	ctx := context.Background()

	t.Run("Fetching non-existent binary from local Beats dir throws an error", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		_, err := downloadAgentBinary(ctx, "foo_fileName", artifact, version)
		assert.NotNil(t, err)
	})

	t.Run("Fetching RPM binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-8.0.0-SNAPSHOT-x86_64.rpm"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(ctx, artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching RPM binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-8.0.0-SNAPSHOT-aarch64.rpm"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(ctx, artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching DEB binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-8.0.0-SNAPSHOT-amd64.deb"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(ctx, artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching DEB binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-8.0.0-SNAPSHOT-arm64.deb"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(ctx, artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching TAR binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-8.0.0-SNAPSHOT-linux-amd64.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(ctx, artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching TAR binary (x86_64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-8.0.0-SNAPSHOT-linux-x86_64.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(ctx, artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching TAR binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-8.0.0-SNAPSHOT-linux-arm64.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(ctx, artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching Docker binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-8.0.0-SNAPSHOT-linux-amd64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(ctx, artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching Docker binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-8.0.0-SNAPSHOT-linux-arm64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(ctx, artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching ubi8 Docker binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-ubi8-8.0.0-SNAPSHOT-linux-amd64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(ctx, artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching ubi8 Docker binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-ubi8-8.0.0-SNAPSHOT-linux-arm64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(ctx, artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
}
