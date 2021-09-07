// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/stretchr/testify/assert"
)

var artifact = "elastic-agent"
var testVersion = "BEATS_VERSION"
var ubi8VersionPrefix = artifact + "-ubi8-" + testVersion
var versionPrefix = artifact + "-" + testVersion

func TestBuildArtifactName(t *testing.T) {
	OS := "linux"
	version := testVersion

	t.Run("For Git commits in version", func(t *testing.T) {
		arch := "x86_64"
		extension := "rpm"
		expectedFileName := "elastic-agent-1.2.3-SNAPSHOT-x86_64.rpm"
		versionWithCommit := "1.2.3-abcdef-SNAPSHOT"

		artifactName := buildArtifactName(artifact, versionWithCommit, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, versionWithCommit, OS, arch, "RPM", false)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For RPM (amd64)", func(t *testing.T) {
		arch := "x86_64"
		extension := "rpm"
		expectedFileName := versionPrefix + "-x86_64.rpm"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "RPM", false)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For RPM (arm64)", func(t *testing.T) {
		arch := "aarch64"
		extension := "rpm"
		expectedFileName := versionPrefix + "-aarch64.rpm"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "RPM", false)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For DEB (amd64)", func(t *testing.T) {
		arch := "amd64"
		extension := "deb"
		expectedFileName := versionPrefix + "-amd64.deb"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "DEB", false)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For DEB (arm64)", func(t *testing.T) {
		arch := "arm64"
		extension := "deb"
		expectedFileName := versionPrefix + "-arm64.deb"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "DEB", false)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For TAR (amd64)", func(t *testing.T) {
		arch := "x86_64"
		extension := "tar.gz"
		expectedFileName := versionPrefix + "-linux-x86_64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", false)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For TAR (arm64)", func(t *testing.T) {
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := versionPrefix + "-linux-arm64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", false)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker from Elastic's repository (amd64)", func(t *testing.T) {
		GithubCommitSha1 = ""
		defer func() { GithubCommitSha1 = "" }()

		artifact = "elastic-agent"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := versionPrefix + "-docker-image-linux-amd64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For Docker from Elastic's repository (arm64)", func(t *testing.T) {
		GithubCommitSha1 = ""
		defer func() { GithubCommitSha1 = "" }()

		artifact = "elastic-agent"
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := versionPrefix + "-docker-image-linux-arm64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker UBI8 from Elastic's repository (amd64)", func(t *testing.T) {
		GithubCommitSha1 = ""
		defer func() { GithubCommitSha1 = "" }()

		artifact = "elastic-agent-ubi8"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := ubi8VersionPrefix + "-docker-image-linux-amd64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For Docker UBI8 from Elastic's repository (arm64)", func(t *testing.T) {
		GithubCommitSha1 = ""
		defer func() { GithubCommitSha1 = "" }()

		artifact = "elastic-agent-ubi8"
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := ubi8VersionPrefix + "-docker-image-linux-arm64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker from local repository (amd64)", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", "/tmp")

		artifact = "elastic-agent"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := versionPrefix + "-linux-amd64.docker.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For Docker from local repository (arm64)", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", "/tmp")

		artifact = "elastic-agent"
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := versionPrefix + "-linux-arm64.docker.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker UBI8 from local repository (amd64)", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", "/tmp")

		artifact = "elastic-agent-ubi8"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := ubi8VersionPrefix + "-linux-amd64.docker.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For Docker UBI8 from local repository (arm64)", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", "/tmp")

		artifact = "elastic-agent-ubi8"
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := ubi8VersionPrefix + "-linux-arm64.docker.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker from GCP (amd64)", func(t *testing.T) {
		GithubCommitSha1 = "0123456789"
		defer func() { GithubCommitSha1 = "" }()

		artifact = "elastic-agent"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := versionPrefix + "-linux-amd64.docker.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For Docker from GCP (arm64)", func(t *testing.T) {
		GithubCommitSha1 = "0123456789"
		defer func() { GithubCommitSha1 = "" }()

		artifact = "elastic-agent"
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := versionPrefix + "-linux-arm64.docker.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker UBI8 from GCP (amd64)", func(t *testing.T) {
		GithubCommitSha1 = "0123456789"
		defer func() { GithubCommitSha1 = "" }()

		artifact = "elastic-agent-ubi8"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := ubi8VersionPrefix + "-linux-amd64.docker.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For Docker UBI8 from GCP (arm64)", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		artifact = "elastic-agent-ubi8"
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := ubi8VersionPrefix + "-linux-arm64.docker.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker for a Pull Request (amd64)", func(t *testing.T) {
		GithubCommitSha1 = "0123456789"
		defer func() { GithubCommitSha1 = "" }()

		artifact = "elastic-agent"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := versionPrefix + "-docker-image-linux-amd64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For Docker for a Pull Request (arm64)", func(t *testing.T) {
		GithubCommitSha1 = "0123456789"
		defer func() { GithubCommitSha1 = "" }()

		artifact = "elastic-agent"
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := versionPrefix + "-docker-image-linux-arm64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
}

func TestCheckPRVersion(t *testing.T) {
	var testVersion = "BEATS_VERSION"

	t.Run("Checking a version should return the version", func(t *testing.T) {
		v := CheckPRVersion(testVersion, testVersion)

		assert.Equal(t, testVersion, v)
	})

	t.Run("A Commit-based version should return base version", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		v := CheckPRVersion(testVersion, testVersion)

		assert.Equal(t, testVersion, v)
	})
}

func TestFetchBeatsBinaryFromLocalPath(t *testing.T) {
	artifact := "elastic-agent"
	beatsDir := path.Join("_testresources", "beats")
	distributionsDir, _ := filepath.Abs(path.Join(beatsDir, "x-pack", "elastic-agent", "build", "distributions"))
	version := testVersion

	ctx := context.Background()

	t.Run("Fetching non-existent binary from local Beats dir throws an error", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		_, err := fetchBeatsBinary(ctx, "foo_fileName", artifact, version, utils.TimeoutFactor, true)
		assert.NotNil(t, err)
	})

	t.Run("Fetching RPM binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := versionPrefix + "-x86_64.rpm"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, utils.TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching RPM binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := versionPrefix + "-aarch64.rpm"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, utils.TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching DEB binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := versionPrefix + "-amd64.deb"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, utils.TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching DEB binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := versionPrefix + "-arm64.deb"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, utils.TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching TAR binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := versionPrefix + "-linux-amd64.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, utils.TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching TAR binary (x86_64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := versionPrefix + "-linux-x86_64.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, utils.TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching TAR binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := versionPrefix + "-linux-arm64.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, utils.TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching Docker binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := versionPrefix + "-linux-amd64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, utils.TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching Docker binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := versionPrefix + "-linux-arm64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, utils.TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching ubi8 Docker binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := ubi8VersionPrefix + "-linux-amd64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, utils.TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching ubi8 Docker binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := ubi8VersionPrefix + "-linux-arm64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, utils.TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
}

func Test_NewElasticVersion(t *testing.T) {
	t.Run("newElasticVersion without git commit", func(t *testing.T) {
		v := newElasticVersion("1.2.3-SNAPSHOT")

		assert.Equal(t, "1.2.3", v.Version, "Version should not include SNAPSHOT nor commit")
		assert.Equal(t, "1.2.3-SNAPSHOT", v.FullVersion, "Full version should include SNAPSHOT")
		assert.Equal(t, "1.2.3", v.HashedVersion, "Hashed version should not include SNAPSHOT")
		assert.Equal(t, "1.2.3-SNAPSHOT", v.SnapshotVersion, "Snapshot version should include SNAPSHOT")
	})

	t.Run("newElasticVersion with git commit", func(t *testing.T) {
		v := newElasticVersion("1.2.3-abcdef-SNAPSHOT")

		assert.Equal(t, "1.2.3", v.Version, "Version should not include SNAPSHOT nor commit")
		assert.Equal(t, "1.2.3-abcdef-SNAPSHOT", v.FullVersion, "Full version should include commit and SNAPSHOT")
		assert.Equal(t, "1.2.3-abcdef", v.HashedVersion, "Hashed version should include commit but no SNAPSHOT")
		assert.Equal(t, "1.2.3-SNAPSHOT", v.SnapshotVersion, "Snapshot version should include SNAPSHOT but no commit")
	})
}

func TestGetGCPBucketCoordinates_Commits(t *testing.T) {
	artifact := "elastic-agent"

	t.Run("Fetching commits bucket for RPM package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "commits/0123456789")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching commits bucket for DEB package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "commits/0123456789")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching commits bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "commits/0123456789")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})

	t.Run("Fetching commits bucket for ubi8 Docker image", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-ubi8-" + testVersion + "-x86_64.tar.gz"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, "elastic-agent-ubi8")
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "commits/0123456789")
		assert.Equal(t, object, "elastic-agent/elastic-agent-ubi8-"+testVersion+"-x86_64.tar.gz")
	})
}

func TestGetGCPBucketCoordinates_Snapshots(t *testing.T) {
	artifact := "elastic-agent"

	t.Run("Fetching snapshots bucket for RPM package", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching snapshots bucket for DEB package", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching snapshots bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}

func Test_GetCommitVersion(t *testing.T) {
	t.Run("GetCommitVersion without git commit", func(t *testing.T) {
		v := GetCommitVersion("1.2.3-SNAPSHOT")

		assert.Equal(t, "1.2.3", v, "Version should not include SNAPSHOT nor commit")
	})

	t.Run("GetCommitVersion with git commit", func(t *testing.T) {
		v := GetCommitVersion("1.2.3-abcdef-SNAPSHOT")

		assert.Equal(t, "1.2.3-abcdef", v, "Version should not include SNAPSHOT nor commit")
	})
}

func Test_GetFullVersion(t *testing.T) {
	t.Run("GetFullVersion without git commit", func(t *testing.T) {
		v := GetFullVersion("1.2.3-SNAPSHOT")

		assert.Equal(t, "1.2.3-SNAPSHOT", v, "Version should not include SNAPSHOT nor commit")
	})

	t.Run("GetFullVersion with git commit", func(t *testing.T) {
		v := GetFullVersion("1.2.3-abcdef-SNAPSHOT")

		assert.Equal(t, "1.2.3-abcdef-SNAPSHOT", v, "Version should not include SNAPSHOT nor commit")
	})
}

func Test_GetSnapshotVersion(t *testing.T) {
	t.Run("GetSnapshotVersion without git commit", func(t *testing.T) {
		v := GetSnapshotVersion("1.2.3-SNAPSHOT")

		assert.Equal(t, "1.2.3-SNAPSHOT", v, "Version should include SNAPSHOT but no commit")
	})

	t.Run("GetCommitVersion with git commit", func(t *testing.T) {
		v := GetSnapshotVersion("1.2.3-abcdef-SNAPSHOT")

		assert.Equal(t, "1.2.3-SNAPSHOT", v, "Version should include SNAPSHOT but no commit")
	})
}

func Test_GetVersion(t *testing.T) {
	t.Run("GetVersion without git commit", func(t *testing.T) {
		v := GetVersion("1.2.3-SNAPSHOT")

		assert.Equal(t, "1.2.3", v, "Version should not include SNAPSHOT nor commit")
	})

	t.Run("GetVersion with git commit", func(t *testing.T) {
		v := GetVersion("1.2.3-abcdef-SNAPSHOT")

		assert.Equal(t, "1.2.3", v, "Version should not include SNAPSHOT nor commit")
	})
}

func TestRemoveCommitFromSnapshot(t *testing.T) {
	assert.Equal(t, "elastic-agent-8.0.0-SNAPSHOT-darwin-x86_64.tar.gz", RemoveCommitFromSnapshot("elastic-agent-8.0.0-abcdef-SNAPSHOT-darwin-x86_64.tar.gz"))
	assert.Equal(t, "8.0.0-SNAPSHOT", RemoveCommitFromSnapshot("8.0.0-a12345-SNAPSHOT"))
	assert.Equal(t, "7.x-SNAPSHOT", RemoveCommitFromSnapshot("7.x-a12345-SNAPSHOT"))
	assert.Equal(t, "7.14.x-SNAPSHOT", RemoveCommitFromSnapshot("7.14.x-a12345-SNAPSHOT"))
	assert.Equal(t, "8.0.0-SNAPSHOT", RemoveCommitFromSnapshot("8.0.0-SNAPSHOT"))
	assert.Equal(t, "7.x-SNAPSHOT", RemoveCommitFromSnapshot("7.x-SNAPSHOT"))
	assert.Equal(t, "7.14.x-SNAPSHOT", RemoveCommitFromSnapshot("7.14.x-SNAPSHOT"))
}

func TestSnapshotHasCommit(t *testing.T) {
	t.Run("Returns true with commits in snapshots", func(t *testing.T) {
		assert.True(t, SnapshotHasCommit("8.0.0-a12345-SNAPSHOT"))
	})

	t.Run("Returns false with commits in snapshots", func(t *testing.T) {
		assert.False(t, SnapshotHasCommit("7.x-SNAPSHOT"))
		assert.False(t, SnapshotHasCommit("8.0.0-SNAPSHOT"))
	})
}
