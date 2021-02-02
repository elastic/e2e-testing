package main

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testVersion = agentVersionBase

func TestDownloadAgentBinary(t *testing.T) {
	artifact := "elastic-agent"
	beatsDir := path.Join("..", "..", "_testresources", "beats")
	distributionsDir := path.Join(beatsDir, "x-pack", "elastic-agent", "build", "distributions")
	OS := "linux"
	version := "8.0.0-SNAPSHOT"

	t.Run("Fetching non-existent binary from local Beats dir throws an error", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		arch := "foo_arch"
		extension := "foo_ext"

		_, _, err := downloadAgentBinary(artifact, version, OS, arch, extension)
		assert.NotNil(t, err)
	})

	t.Run("Fetching RPM binary from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		arch := "x86_64"
		extension := "rpm"
		expectedFileName := "elastic-agent-8.0.0-SNAPSHOT-x86_64.rpm"

		newFileName, downloadedFilePath, err := downloadAgentBinary(artifact, version, OS, arch, extension)
		assert.Nil(t, err)
		assert.Equal(t, newFileName, expectedFileName)
		assert.Equal(t, downloadedFilePath, path.Join(distributionsDir, expectedFileName))
	})

	t.Run("Fetching DEB binary from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		arch := "amd64"
		extension := "deb"
		expectedFileName := "elastic-agent-8.0.0-SNAPSHOT-amd64.deb"

		newFileName, downloadedFilePath, err := downloadAgentBinary(artifact, version, OS, arch, extension)
		assert.Nil(t, err)
		assert.Equal(t, newFileName, expectedFileName)
		assert.Equal(t, downloadedFilePath, path.Join(distributionsDir, expectedFileName))
	})

	t.Run("Fetching TAR binary from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-8.0.0-SNAPSHOT-linux-amd64.tar.gz"

		newFileName, downloadedFilePath, err := downloadAgentBinary(artifact, version, OS, arch, extension)
		assert.Nil(t, err)
		assert.Equal(t, newFileName, expectedFileName)
		assert.Equal(t, downloadedFilePath, path.Join(distributionsDir, expectedFileName))
	})
}

func TestGetGCPBucketCoordinates_Commits(t *testing.T) {
	artifact := "elastic-agent"
	version := testVersion

	t.Run("Fetching commits bucket for RPM package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "commits/0123456789")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching commits bucket for DEB package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "commits/0123456789")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching commits bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "commits/0123456789")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}

func TestGetGCPBucketCoordinates_CommitsForAPullRequest(t *testing.T) {
	artifact := "elastic-agent"
	version := "pr-23456"

	t.Run("Fetching commits bucket for RPM package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching commits bucket for DEB package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching commits bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}

func TestGetGCPBucketCoordinates_PullRequests(t *testing.T) {
	artifact := "elastic-agent"
	version := "pr-23456"

	t.Run("Fetching commits bucket for RPM package", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching commits bucket for DEB package", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching commits bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}

func TestGetGCPBucketCoordinates_Snapshots(t *testing.T) {
	artifact := "elastic-agent"
	version := testVersion

	t.Run("Fetching commits bucket for RPM package", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching commits bucket for DEB package", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching commits bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}
