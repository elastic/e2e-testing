// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
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

	t.Run("Fetching non-existent binary from local Beats dir throws an error", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		_, err := downloadAgentBinary("foo_fileName", artifact, version)
		assert.NotNil(t, err)
	})

	t.Run("Fetching RPM binary from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-8.0.0-SNAPSHOT-x86_64.rpm"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching DEB binary from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-8.0.0-SNAPSHOT-amd64.deb"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching TAR binary from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-8.0.0-SNAPSHOT-linux-amd64.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching Docker binary from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-8.0.0-SNAPSHOT-linux-amd64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching ubi8 Docker binary from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-ubi8-8.0.0-SNAPSHOT-linux-amd64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := downloadAgentBinary(artifactName, artifact, version)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
}
