package utils

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	gabs "github.com/Jeffail/gabs/v2"
)

var artifact = "elastic-agent"
var testVersion = "BEATS_VERSION"
var ubi8VersionPrefix = artifact + "-ubi8-" + testVersion
var versionPrefix = artifact + "-" + testVersion

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

func TestDownloadFile(t *testing.T) {
	f, err := DownloadFile("https://www.elastic.co/robots.txt")
	assert.Nil(t, err)
	defer os.Remove(filepath.Dir(f))
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

	t.Run("Returns false with commits in snapshots", func(t *testing.T) {
		assert.False(t, IsCommit("8.0.0-a12345-SNAPSHOT"))
	})
}

func TestProcessBucketSearchPage_CommitFound(t *testing.T) {
	// retrieving last element in commits.json
	object := "024b732844d40bdb2bf806480af2b03fcb8fbdbe/elastic-agent/" + versionPrefix + "-darwin-x86_64.tar.gz"

	mediaLink, err := processBucketSearchPage(commitsJSON, 1, bucket, commits, object)
	assert.Nil(t, err)
	assert.True(t, mediaLink == "https://storage.googleapis.com/download/storage/v1/b/beats-ci-artifacts/o/commits%2F024b732844d40bdb2bf806480af2b03fcb8fbdbe%2Felastic-agent%2F"+versionPrefix+"-darwin-x86_64.tar.gz?generation=1612983859986704&alt=media")
}

func TestProcessBucketSearchPage_CommitsNotFound(t *testing.T) {
	object := "foo/" + versionPrefix + "-linux-amd64.docker.tar.gz"

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
