package utils

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	gabs "github.com/Jeffail/gabs/v2"
)

var testVersion = "7.14.0-SNAPSHOT"

const bucket = "beats-ci-artifacts"
const commits = "commits"
const snapshots = "snapshots"

var nextTokenParamJSON *gabs.Container
var commitsJSON *gabs.Container
var snapshotsJSON *gabs.Container

func init() {
	nextTokenParamContent, err := ioutil.ReadFile(path.Join("..", "_testresources", "gcp", "nextPageParam.json"))
	if err != nil {
		os.Exit(1)
	}
	nextTokenParamJSON, _ = gabs.ParseJSON([]byte(nextTokenParamContent))

	commitsContent, err := ioutil.ReadFile(path.Join("..", "_testresources", "gcp", "commits.json"))
	if err != nil {
		os.Exit(1)
	}
	commitsJSON, _ = gabs.ParseJSON([]byte(commitsContent))

	snapshotsContent, err := ioutil.ReadFile(path.Join("..", "_testresources", "gcp", "snapshots.json"))
	if err != nil {
		os.Exit(1)
	}
	snapshotsJSON, _ = gabs.ParseJSON([]byte(snapshotsContent))
}

func TestBuildArtifactName(t *testing.T) {
	artifact := "elastic-agent"
	OS := "linux"
	version := "7.14.0-SNAPSHOT"

	t.Run("For RPM (amd64)", func(t *testing.T) {
		arch := "x86_64"
		extension := "rpm"
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-x86_64.rpm"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "RPM", false)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For RPM (arm64)", func(t *testing.T) {
		arch := "aarch64"
		extension := "rpm"
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-aarch64.rpm"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "RPM", false)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For DEB (amd64)", func(t *testing.T) {
		arch := "amd64"
		extension := "deb"
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-amd64.deb"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "DEB", false)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For DEB (arm64)", func(t *testing.T) {
		arch := "arm64"
		extension := "deb"
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-arm64.deb"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "DEB", false)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For TAR (amd64)", func(t *testing.T) {
		arch := "x86_64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-linux-x86_64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", false)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For TAR (arm64)", func(t *testing.T) {
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-linux-arm64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", false)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker from Elastic's repository (amd64)", func(t *testing.T) {
		defer os.Unsetenv("BEATS_USE_CI_SNAPSHOTS")
		os.Setenv("BEATS_USE_CI_SNAPSHOTS", "false")

		artifact = "elastic-agent"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-docker-image-linux-amd64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For Docker from Elastic's repository (arm64)", func(t *testing.T) {
		defer os.Unsetenv("BEATS_USE_CI_SNAPSHOTS")
		os.Setenv("BEATS_USE_CI_SNAPSHOTS", "false")

		artifact = "elastic-agent"
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-docker-image-linux-arm64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker UBI8 from Elastic's repository (amd64)", func(t *testing.T) {
		defer os.Unsetenv("BEATS_USE_CI_SNAPSHOTS")
		os.Setenv("BEATS_USE_CI_SNAPSHOTS", "false")

		artifact = "elastic-agent-ubi8"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-ubi8-7.14.0-SNAPSHOT-docker-image-linux-amd64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For Docker UBI8 from Elastic's repository (arm64)", func(t *testing.T) {
		defer os.Unsetenv("BEATS_USE_CI_SNAPSHOTS")
		os.Setenv("BEATS_USE_CI_SNAPSHOTS", "false")

		artifact = "elastic-agent-ubi8"
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-ubi8-7.14.0-SNAPSHOT-docker-image-linux-arm64.tar.gz"

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
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-linux-amd64.docker.tar.gz"

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
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-linux-arm64.docker.tar.gz"

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
		expectedFileName := "elastic-agent-ubi8-7.14.0-SNAPSHOT-linux-amd64.docker.tar.gz"

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
		expectedFileName := "elastic-agent-ubi8-7.14.0-SNAPSHOT-linux-arm64.docker.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker from GCP (amd64)", func(t *testing.T) {
		defer os.Unsetenv("BEATS_USE_CI_SNAPSHOTS")
		os.Setenv("BEATS_USE_CI_SNAPSHOTS", "true")

		artifact = "elastic-agent"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-linux-amd64.docker.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For Docker from GCP (arm64)", func(t *testing.T) {
		defer os.Unsetenv("BEATS_USE_CI_SNAPSHOTS")
		os.Setenv("BEATS_USE_CI_SNAPSHOTS", "true")

		artifact = "elastic-agent"
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-linux-arm64.docker.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker UBI8 from GCP (amd64)", func(t *testing.T) {
		defer os.Unsetenv("BEATS_USE_CI_SNAPSHOTS")
		os.Setenv("BEATS_USE_CI_SNAPSHOTS", "true")

		artifact = "elastic-agent-ubi8"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-ubi8-7.14.0-SNAPSHOT-linux-amd64.docker.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For Docker UBI8 from GCP (arm64)", func(t *testing.T) {
		defer os.Unsetenv("BEATS_USE_CI_SNAPSHOTS")
		os.Setenv("BEATS_USE_CI_SNAPSHOTS", "true")

		artifact = "elastic-agent-ubi8"
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-ubi8-7.14.0-SNAPSHOT-linux-arm64.docker.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker for a Pull Request (amd64)", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		artifact = "elastic-agent"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-docker-image-linux-amd64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
	t.Run("For Docker for a Pull Request (arm64)", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		artifact = "elastic-agent"
		arch := "arm64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-7.14.0-SNAPSHOT-docker-image-linux-arm64.tar.gz"

		artifactName := buildArtifactName(artifact, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = buildArtifactName(artifact, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
}

func TestCheckPRVersion(t *testing.T) {
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

func TestDownloadFile(t *testing.T) {
	f, err := DownloadFile("https://www.elastic.co/robots.txt")
	assert.Nil(t, err)
	defer os.Remove(filepath.Dir(f))

	assert.True(t, strings.HasSuffix(f, "robots.txt"))
}

func TestFetchBeatsBinaryFromLocalPath(t *testing.T) {
	artifact := "elastic-agent"
	beatsDir := path.Join("..", "_testresources", "beats")
	distributionsDir, _ := filepath.Abs(path.Join(beatsDir, "x-pack", "elastic-agent", "build", "distributions"))
	version := "7.14.0-SNAPSHOT"

	ctx := context.Background()

	t.Run("Fetching non-existent binary from local Beats dir throws an error", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		_, err := fetchBeatsBinary(ctx, "foo_fileName", artifact, version, TimeoutFactor, true)
		assert.NotNil(t, err)
	})

	t.Run("Fetching RPM binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-7.14.0-SNAPSHOT-x86_64.rpm"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching RPM binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-7.14.0-SNAPSHOT-aarch64.rpm"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching DEB binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-7.14.0-SNAPSHOT-amd64.deb"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching DEB binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-7.14.0-SNAPSHOT-arm64.deb"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching TAR binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-7.14.0-SNAPSHOT-linux-amd64.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching TAR binary (x86_64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-7.14.0-SNAPSHOT-linux-x86_64.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching TAR binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-7.14.0-SNAPSHOT-linux-arm64.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching Docker binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-7.14.0-SNAPSHOT-linux-amd64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching Docker binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-7.14.0-SNAPSHOT-linux-arm64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})

	t.Run("Fetching ubi8 Docker binary (amd64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-ubi8-7.14.0-SNAPSHOT-linux-amd64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
	t.Run("Fetching ubi8 Docker binary (arm64) from local Beats dir", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", beatsDir)

		artifactName := "elastic-agent-ubi8-7.14.0-SNAPSHOT-linux-arm64.docker.tar.gz"
		expectedFilePath := path.Join(distributionsDir, artifactName)

		downloadedFilePath, err := fetchBeatsBinary(ctx, artifactName, artifact, version, TimeoutFactor, true)
		assert.Nil(t, err)
		assert.Equal(t, downloadedFilePath, expectedFilePath)
	})
}

func TestGetArchitecture(t *testing.T) {
	t.Run("Retrieving amd architecture", func(t *testing.T) {
		fallbackArch := os.Getenv("GOARCH")
		os.Setenv("GOARCH", "amd64")
		defer os.Setenv("GOARCH", fallbackArch)

		assert.Equal(t, "amd64", GetArchitecture())
	})

	t.Run("Retrieving amd architecture as fallback", func(t *testing.T) {
		fallbackArch := os.Getenv("GOARCH")
		os.Setenv("GOARCH", "arch-not-found")
		defer os.Setenv("GOARCH", fallbackArch)

		assert.Equal(t, "amd64", GetArchitecture())
	})

	t.Run("Retrieving arm architecture", func(t *testing.T) {
		fallbackArch := os.Getenv("GOARCH")
		os.Setenv("GOARCH", "arm64")
		defer os.Setenv("GOARCH", fallbackArch)

		assert.Equal(t, "arm64", GetArchitecture())
	})
}

func TestGetBucketSearchNextPageParam_HasMorePages(t *testing.T) {
	expectedParam := "&pageToken=foo"

	param := getBucketSearchNextPageParam(nextTokenParamJSON)
	assert.True(t, param == expectedParam)
}

func TestGetBucketSearchNextPageParam_HasNoMorePages(t *testing.T) {
	// this JSON file does not contain the tokken field
	param := getBucketSearchNextPageParam(commitsJSON)
	assert.True(t, param == "")
}

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

func TestGetGCPBucketCoordinates_Snapshots(t *testing.T) {
	artifact := "elastic-agent"
	version := testVersion

	t.Run("Fetching snapshots bucket for RPM package", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching snapshots bucket for DEB package", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching snapshots bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}

func TestIsCommit(t *testing.T) {
	t.Run("Returns true with commits", func(t *testing.T) {
		assert.True(t, IsCommit("abcdef1234"))
		assert.True(t, IsCommit("a12345"))
		assert.True(t, IsCommit("abcdef1"))
	})

	t.Run("Returns false with non-commits", func(t *testing.T) {
		assert.False(t, IsCommit("master"))
		assert.False(t, IsCommit("7.x"))
		assert.False(t, IsCommit("7.12.x"))
		assert.False(t, IsCommit("7.11.x"))
		assert.False(t, IsCommit("pr12345"))
	})
}

func TestProcessBucketSearchPage_CommitFound(t *testing.T) {
	// retrieving last element in commits.json
	object := "024b732844d40bdb2bf806480af2b03fcb8fbdbe/elastic-agent/elastic-agent-7.14.0-SNAPSHOT-darwin-x86_64.tar.gz"

	mediaLink, err := processBucketSearchPage(commitsJSON, 1, bucket, commits, object)
	assert.Nil(t, err)
	assert.True(t, mediaLink == "https://storage.googleapis.com/download/storage/v1/b/beats-ci-artifacts/o/commits%2F024b732844d40bdb2bf806480af2b03fcb8fbdbe%2Felastic-agent%2Felastic-agent-7.14.0-SNAPSHOT-darwin-x86_64.tar.gz?generation=1612983859986704&alt=media")
}

func TestProcessBucketSearchPage_CommitsNotFound(t *testing.T) {
	object := "foo/elastic-agent-7.14.0-SNAPSHOT-linux-amd64.docker.tar.gz"

	mediaLink, err := processBucketSearchPage(commitsJSON, 1, bucket, commits, object)
	assert.NotNil(t, err)
	assert.True(t, mediaLink == "")
}

func TestProcessBucketSearchPage_SnapshotsFound(t *testing.T) {
	// retrieving last element in snapshots.json
	object := "filebeat/filebeat-oss-7.10.2-SNAPSHOT-arm64.deb"

	mediaLink, err := processBucketSearchPage(snapshotsJSON, 1, bucket, snapshots, object)
	assert.Nil(t, err)
	assert.True(t, mediaLink == "https://storage.googleapis.com/download/storage/v1/b/beats-ci-artifacts/o/snapshots%2Ffilebeat%2Ffilebeat-oss-7.10.2-SNAPSHOT-arm64.deb?generation=1610629747796392&alt=media")
}

func TestProcessBucketSearchPage_SnapshotsNotFound(t *testing.T) {
	object := "filebeat/filebeat-oss-7.12.2-SNAPSHOT-arm64.deb"

	mediaLink, err := processBucketSearchPage(snapshotsJSON, 1, bucket, snapshots, object)
	assert.NotNil(t, err)
	assert.True(t, mediaLink == "")
}
