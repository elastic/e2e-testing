package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testVersion = agentVersionBase

func TestGetGCPBucketCoordinates_Commits(t *testing.T) {
	artifact := "elastic-agent"
	version := testVersion
	OS := "linux"

	t.Run("Fetching commits bucket for RPM package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		arch := "x86_64"
		extension := "rpm"
		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		newFileName, bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, OS, arch, extension)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "commits/0123456789")
		assert.Equal(t, newFileName, "elastic-agent-"+testVersion+"-x86_64.rpm")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching commits bucket for DEB package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		arch := "amd64"
		extension := "deb"
		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		newFileName, bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, OS, arch, extension)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "commits/0123456789")
		assert.Equal(t, newFileName, "elastic-agent-"+testVersion+"-amd64.deb")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching commits bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		arch := "x86_64"
		extension := "tar.gz"
		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		newFileName, bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, OS, arch, extension)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "commits/0123456789")
		assert.Equal(t, newFileName, "elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}

func TestGetGCPBucketCoordinates_CommitsForAPullRequest(t *testing.T) {
	artifact := "elastic-agent"
	version := "pr-23456"
	OS := "linux"

	t.Run("Fetching commits bucket for RPM package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		arch := "x86_64"
		extension := "rpm"
		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		newFileName, bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, OS, arch, extension)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, newFileName, "elastic-agent-"+testVersion+"-x86_64.rpm")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching commits bucket for DEB package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		arch := "amd64"
		extension := "deb"
		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		newFileName, bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, OS, arch, extension)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, newFileName, "elastic-agent-"+testVersion+"-amd64.deb")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching commits bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		arch := "x86_64"
		extension := "tar.gz"
		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		newFileName, bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, OS, arch, extension)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, newFileName, "elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}

func TestGetGCPBucketCoordinates_PullRequests(t *testing.T) {
	artifact := "elastic-agent"
	version := "pr-23456"
	OS := "linux"

	t.Run("Fetching commits bucket for RPM package", func(t *testing.T) {
		arch := "x86_64"
		extension := "rpm"
		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		newFileName, bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, OS, arch, extension)
		assert.Equal(t, newFileName, "elastic-agent-"+testVersion+"-x86_64.rpm")
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching commits bucket for DEB package", func(t *testing.T) {
		arch := "amd64"
		extension := "deb"
		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		newFileName, bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, OS, arch, extension)
		assert.Equal(t, newFileName, "elastic-agent-"+testVersion+"-amd64.deb")
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching commits bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		arch := "x86_64"
		extension := "tar.gz"
		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		newFileName, bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, OS, arch, extension)
		assert.Equal(t, newFileName, "elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}

func TestGetGCPBucketCoordinates_Snapshots(t *testing.T) {
	artifact := "elastic-agent"
	version := testVersion
	OS := "linux"

	t.Run("Fetching commits bucket for RPM package", func(t *testing.T) {
		arch := "x86_64"
		extension := "rpm"
		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		newFileName, bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, OS, arch, extension)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, newFileName, "elastic-agent-"+testVersion+"-x86_64.rpm")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching commits bucket for DEB package", func(t *testing.T) {
		arch := "amd64"
		extension := "deb"
		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		newFileName, bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, OS, arch, extension)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, newFileName, "elastic-agent-"+testVersion+"-amd64.deb")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching commits bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		arch := "x86_64"
		extension := "tar.gz"
		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		newFileName, bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, OS, arch, extension)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, newFileName, "elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}
