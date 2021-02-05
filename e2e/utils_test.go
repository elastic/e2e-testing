package e2e

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	gabs "github.com/Jeffail/gabs/v2"
)

var testVersion = "8.0.0-SNAPSHOT"

const bucket = "beats-ci-artifacts"
const pullRequests = "pull-requests"
const snapshots = "snapshots"

var nextTokenParamJSON *gabs.Container
var pullRequestsJSON *gabs.Container
var snapshotsJSON *gabs.Container

func init() {
	nextTokenParamContent, err := ioutil.ReadFile("_testresources/gcp/nextPageParam.json")
	if err != nil {
		os.Exit(1)
	}
	nextTokenParamJSON, _ = gabs.ParseJSON([]byte(nextTokenParamContent))

	pullRequestsContent, err := ioutil.ReadFile("_testresources/gcp/pull-requests.json")
	if err != nil {
		os.Exit(1)
	}
	pullRequestsJSON, _ = gabs.ParseJSON([]byte(pullRequestsContent))

	snapshotsContent, err := ioutil.ReadFile("_testresources/gcp/snapshots.json")
	if err != nil {
		os.Exit(1)
	}
	snapshotsJSON, _ = gabs.ParseJSON([]byte(snapshotsContent))
}

func TestBuildArtifactName(t *testing.T) {
	artifact := "elastic-agent"
	OS := "linux"
	version := "8.0.0-SNAPSHOT"

	t.Run("For RPM", func(t *testing.T) {
		arch := "x86_64"
		extension := "rpm"
		expectedFileName := "elastic-agent-8.0.0-SNAPSHOT-x86_64.rpm"

		artifactName := BuildArtifactName(artifact, version, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = BuildArtifactName(artifact, version, version, OS, arch, "RPM", false)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For DEB", func(t *testing.T) {
		arch := "amd64"
		extension := "deb"
		expectedFileName := "elastic-agent-8.0.0-SNAPSHOT-amd64.deb"

		artifactName := BuildArtifactName(artifact, version, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = BuildArtifactName(artifact, version, version, OS, arch, "DEB", false)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For TAR", func(t *testing.T) {
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-8.0.0-SNAPSHOT-linux-amd64.tar.gz"

		artifactName := BuildArtifactName(artifact, version, version, OS, arch, extension, false)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = BuildArtifactName(artifact, version, version, OS, arch, "TAR.GZ", false)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker from Elastic's repository", func(t *testing.T) {
		defer os.Unsetenv("BEATS_USE_CI_SNAPSHOTS")
		os.Setenv("BEATS_USE_CI_SNAPSHOTS", "false")

		artifact = "elastic-agent"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-8.0.0-SNAPSHOT-docker-image-linux-amd64.tar.gz"

		artifactName := BuildArtifactName(artifact, version, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = BuildArtifactName(artifact, version, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker UBI8 from Elastic's repository", func(t *testing.T) {
		defer os.Unsetenv("BEATS_USE_CI_SNAPSHOTS")
		os.Setenv("BEATS_USE_CI_SNAPSHOTS", "false")

		artifact = "elastic-agent-ubi8"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-ubi8-8.0.0-SNAPSHOT-docker-image-linux-amd64.tar.gz"

		artifactName := BuildArtifactName(artifact, version, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = BuildArtifactName(artifact, version, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker from local repository", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", "/tmp")

		artifact = "elastic-agent"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-8.0.0-SNAPSHOT-linux-amd64.docker.tar.gz"

		artifactName := BuildArtifactName(artifact, version, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = BuildArtifactName(artifact, version, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker UBI8 from local repository", func(t *testing.T) {
		defer os.Unsetenv("BEATS_LOCAL_PATH")
		os.Setenv("BEATS_LOCAL_PATH", "/tmp")

		artifact = "elastic-agent-ubi8"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-ubi8-8.0.0-SNAPSHOT-linux-amd64.docker.tar.gz"

		artifactName := BuildArtifactName(artifact, version, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = BuildArtifactName(artifact, version, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker from GCP", func(t *testing.T) {
		defer os.Unsetenv("BEATS_USE_CI_SNAPSHOTS")
		os.Setenv("BEATS_USE_CI_SNAPSHOTS", "true")

		artifact = "elastic-agent"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-8.0.0-SNAPSHOT-linux-amd64.docker.tar.gz"

		artifactName := BuildArtifactName(artifact, version, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = BuildArtifactName(artifact, version, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker UBI8 from GCP", func(t *testing.T) {
		defer os.Unsetenv("BEATS_USE_CI_SNAPSHOTS")
		os.Setenv("BEATS_USE_CI_SNAPSHOTS", "true")

		artifact = "elastic-agent-ubi8"
		arch := "amd64"
		extension := "tar.gz"
		expectedFileName := "elastic-agent-ubi8-8.0.0-SNAPSHOT-linux-amd64.docker.tar.gz"

		artifactName := BuildArtifactName(artifact, version, version, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = BuildArtifactName(artifact, version, version, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})

	t.Run("For Docker for a Pull Request", func(t *testing.T) {
		defer os.Unsetenv("ELASTIC_AGENT_VERSION")
		os.Setenv("ELASTIC_AGENT_VERSION", "pr-12345")

		artifact = "elastic-agent"
		arch := "amd64"
		extension := "tar.gz"
		fallbackVersion := "8.0.0-SNAPSHOT"
		expectedFileName := "elastic-agent-8.0.0-SNAPSHOT-docker-image-linux-amd64.tar.gz"

		artifactName := BuildArtifactName(artifact, version, fallbackVersion, OS, arch, extension, true)
		assert.Equal(t, expectedFileName, artifactName)

		artifactName = BuildArtifactName(artifact, version, fallbackVersion, OS, arch, "TAR.GZ", true)
		assert.Equal(t, expectedFileName, artifactName)
	})
}

func TestGetBucketSearchNextPageParam_HasMorePages(t *testing.T) {
	expectedParam := "&pageToken=foo"

	param := getBucketSearchNextPageParam(nextTokenParamJSON)
	assert.True(t, param == expectedParam)
}

func TestGetBucketSearchNextPageParam_HasNoMorePages(t *testing.T) {
	// this JSON file does not contain the tokken field
	param := getBucketSearchNextPageParam(pullRequestsJSON)
	assert.True(t, param == "")
}

func TestGetGCPBucketCoordinates_Commits(t *testing.T) {
	artifact := "elastic-agent"
	version := testVersion

	t.Run("Fetching commits bucket for RPM package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, testVersion)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "commits/0123456789")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching commits bucket for DEB package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, testVersion)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "commits/0123456789")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching commits bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, testVersion)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "commits/0123456789")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}

func TestGetGCPBucketCoordinates_CommitsForAPullRequest(t *testing.T) {
	artifact := "elastic-agent"
	version := "pr-23456"

	t.Run("Fetching pull request bucket for RPM package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, testVersion)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching pull request bucket for DEB package", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, testVersion)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching pull request bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		defer os.Unsetenv("GITHUB_CHECK_SHA1")
		os.Setenv("GITHUB_CHECK_SHA1", "0123456789")

		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, testVersion)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}

func TestGetGCPBucketCoordinates_PullRequests(t *testing.T) {
	artifact := "elastic-agent"
	version := "pr-23456"

	t.Run("Fetching pull request bucket for RPM package", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, testVersion)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching pull request bucket for DEB package", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, testVersion)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching pull request bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, testVersion)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "pull-requests/pr-23456")
		assert.Equal(t, object, "elastic-agent/elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}

func TestGetGCPBucketCoordinates_Snapshots(t *testing.T) {
	artifact := "elastic-agent"
	version := testVersion

	t.Run("Fetching snapshots bucket for RPM package", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-x86_64.rpm"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, testVersion)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-x86_64.rpm")
	})

	t.Run("Fetching snapshots bucket for DEB package", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-amd64.deb"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, testVersion)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-amd64.deb")
	})

	t.Run("Fetching snapshots bucket for TAR package adds OS to fileName and object", func(t *testing.T) {
		fileName := "elastic-agent-" + testVersion + "-linux-x86_64.tar.gz"

		bucket, prefix, object := getGCPBucketCoordinates(fileName, artifact, version, testVersion)
		assert.Equal(t, bucket, "beats-ci-artifacts")
		assert.Equal(t, prefix, "snapshots/elastic-agent")
		assert.Equal(t, object, "elastic-agent-"+testVersion+"-linux-x86_64.tar.gz")
	})
}

func TestProcessBucketSearchPage_PullRequestsFound(t *testing.T) {
	// retrieving last element in pull-requests.json
	object := "pr-22495/filebeat/filebeat-8.0.0-SNAPSHOT-linux-amd64.docker.tar.gz"

	mediaLink, err := processBucketSearchPage(pullRequestsJSON, 1, bucket, pullRequests, object)
	assert.Nil(t, err)
	assert.True(t, mediaLink == "https://storage.googleapis.com/download/storage/v1/b/beats-ci-artifacts/o/pull-requests%2Fpr-22495%2Ffilebeat%2Ffilebeat-8.0.0-SNAPSHOT-linux-amd64.docker.tar.gz?generation=1610634841693588&alt=media")
}

func TestProcessBucketSearchPage_PullRequestsNotFound(t *testing.T) {
	object := "pr-fooo/filebeat/filebeat-8.0.0-SNAPSHOT-linux-amd64.docker.tar.gz"

	mediaLink, err := processBucketSearchPage(pullRequestsJSON, 1, bucket, pullRequests, object)
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
