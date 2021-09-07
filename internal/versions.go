// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/curl"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	"go.elastic.co/apm"

	log "github.com/sirupsen/logrus"
)

// to avoid downloading the same artifacts, we are adding this map to cache the URL of the downloaded binaries, using as key
// the URL of the artifact. If another installer is trying to download the same URL, it will return the location of the
// already downloaded artifact.
var binariesCache = map[string]string{}

// to avoid fetching the same Elastic artifacts version, we are adding this map to cache the version of the Elastic artifacts,
// using as key the URL of the version. If another request is trying to fetch the same URL, it will return the string version
// of the already requested one.
var elasticVersionsCache = map[string]string{}

// elasticVersion represents a version
type elasticVersion struct {
	Version         string // 8.0.0
	FullVersion     string // 8.0.0-abcdef-SNAPSHOT
	HashedVersion   string // 8.0.0-abcdef
	SnapshotVersion string // 8.0.0-SNAPSHOT
}

func newElasticVersion(version string) *elasticVersion {
	versionWithoutCommit := RemoveCommitFromSnapshot(version)
	versionWithoutSnapshot := strings.ReplaceAll(version, "-SNAPSHOT", "")
	versionWithoutCommitAndSnapshot := strings.ReplaceAll(versionWithoutCommit, "-SNAPSHOT", "")

	return &elasticVersion{
		FullVersion:     version,
		HashedVersion:   versionWithoutSnapshot,
		SnapshotVersion: versionWithoutCommit,
		Version:         versionWithoutCommitAndSnapshot,
	}
}

// CheckPRVersion returns a fallback version if the version comes from a commit
func CheckPRVersion(version string, fallbackVersion string) string {
	commitSHA := shell.GetEnv("GITHUB_CHECK_SHA1", "")
	if commitSHA != "" {
		return fallbackVersion
	}

	return version
}

// FetchElasticArtifact fetches an artifact from the right repository, returning binary name, path and error
func FetchElasticArtifact(ctx context.Context, artifact string, version string, os string, arch string, extension string, isDocker bool, xpack bool) (string, string, error) {
	binaryName := buildArtifactName(artifact, version, os, arch, extension, isDocker)
	binaryPath, err := fetchBeatsBinary(ctx, binaryName, artifact, version, utils.TimeoutFactor, xpack)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":  artifact,
			"version":   version,
			"os":        os,
			"arch":      arch,
			"extension": extension,
			"error":     err,
		}).Error("Could not download the binary for the agent")
		return "", "", err
	}

	return binaryName, binaryPath, nil
}

// GetCommitVersion returns a version including the version and the git commit, if it exists
func GetCommitVersion(version string) string {
	return newElasticVersion(version).HashedVersion
}

// GetElasticArtifactURL returns the URL of a released artifact, which its full name is defined in the first argument,
// from Elastic's artifact repository, building the JSON path query based on the full name
// i.e. GetElasticArtifactURL("elastic-agent-$VERSION-$ARCH.deb", "elastic-agent", "$VERSION")
// i.e. GetElasticArtifactURL("elastic-agent-$VERSION-x86_64.rpm", "elastic-agent","$VERSION")
// i.e. GetElasticArtifactURL("elastic-agent-$VERSION-linux-$ARCH.tar.gz", "elastic-agent","$VERSION")
func GetElasticArtifactURL(artifactName string, artifact string, version string) (string, error) {
	exp := utils.GetExponentialBackOff(time.Minute)

	retryCount := 1

	body := ""

	tmpVersion := version
	hasCommit := SnapshotHasCommit(version)
	if hasCommit {
		log.WithFields(log.Fields{
			"version": version,
		}).Trace("Removing SNAPSHOT from version including commit")

		// remove the SNAPSHOT from the VERSION as the artifacts API supports commits in the version, but without the snapshot suffix
		tmpVersion = GetCommitVersion(version)
	}

	apiStatus := func() error {
		r := curl.HTTPRequest{
			URL: fmt.Sprintf("https://artifacts-api.elastic.co/v1/search/%s/%s?x-elastic-no-kpi=true", tmpVersion, artifact),
		}

		response, err := curl.Get(r)
		if err != nil {
			log.WithFields(log.Fields{
				"artifact":       artifact,
				"artifactName":   artifactName,
				"version":        tmpVersion,
				"error":          err,
				"retry":          retryCount,
				"statusEndpoint": r.URL,
				"elapsedTime":    exp.GetElapsedTime(),
			}).Warn("The Elastic artifacts API is not available yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"retries":        retryCount,
			"statusEndpoint": r.URL,
			"elapsedTime":    exp.GetElapsedTime(),
		}).Debug("The Elastic artifacts API is available")

		body = response
		return nil
	}

	err := backoff.Retry(apiStatus, exp)
	if err != nil {
		return "", err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":     artifact,
			"artifactName": artifactName,
			"version":      tmpVersion,
		}).Error("Could not parse the response body for the artifact")
		return "", err
	}

	log.WithFields(log.Fields{
		"retries":      retryCount,
		"artifact":     artifact,
		"artifactName": artifactName,
		"elapsedTime":  exp.GetElapsedTime(),
		"version":      tmpVersion,
	}).Trace("Artifact found")

	if hasCommit {
		// remove commit from the artifact as it comes like this: elastic-agent-8.0.0-abcdef-SNAPSHOT-darwin-x86_64.tar.gz
		artifactName = RemoveCommitFromSnapshot(artifactName)
	}

	packagesObject := jsonParsed.Path("packages")
	// we need to get keys with dots using Search instead of Path
	downloadObject := packagesObject.Search(artifactName)
	downloadURL := downloadObject.Path("url").Data().(string)

	return downloadURL, nil
}

// GetElasticArtifactVersion returns the current version:
// 1. Elastic's artifact repository, building the JSON path query based
// If the version is a SNAPSHOT including a commit, then it will directly use the version without checking the artifacts API
// i.e. GetElasticArtifactVersion("$VERSION-abcdef-SNAPSHOT")
func GetElasticArtifactVersion(version string) (string, error) {
	cacheKey := fmt.Sprintf("https://artifacts-api.elastic.co/v1/versions/%s/?x-elastic-no-kpi=true", version)

	if val, ok := elasticVersionsCache[cacheKey]; ok {
		log.WithFields(log.Fields{
			"URL":     cacheKey,
			"version": val,
		}).Debug("Retrieving version from local cache")
		return val, nil
	}

	if SnapshotHasCommit(version) {
		elasticVersionsCache[cacheKey] = version
		return version, nil
	}

	exp := utils.GetExponentialBackOff(time.Minute)

	retryCount := 1

	body := ""

	apiStatus := func() error {
		r := curl.HTTPRequest{
			URL: cacheKey,
		}

		response, err := curl.Get(r)
		if err != nil {
			log.WithFields(log.Fields{
				"version":        version,
				"error":          err,
				"retry":          retryCount,
				"statusEndpoint": r.URL,
				"elapsedTime":    exp.GetElapsedTime(),
			}).Warn("The Elastic artifacts API is not available yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"retries":        retryCount,
			"statusEndpoint": r.URL,
			"elapsedTime":    exp.GetElapsedTime(),
		}).Debug("The Elastic artifacts API is available")

		body = response
		return nil
	}

	err := backoff.Retry(apiStatus, exp)
	if err != nil {
		return "", err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": version,
		}).Error("Could not parse the response body to retrieve the version")
		return "", err
	}

	builds := jsonParsed.Path("version.builds")

	lastBuild := builds.Children()[0]
	latestVersion := lastBuild.Path("version").Data().(string)

	log.WithFields(log.Fields{
		"alias":   version,
		"version": latestVersion,
	}).Debug("Latest version for current version obtained")

	elasticVersionsCache[cacheKey] = latestVersion

	return latestVersion, nil
}

// GetFullVersion returns a version including the full version: version, git commit and snapshot
func GetFullVersion(version string) string {
	return newElasticVersion(version).FullVersion
}

// GetSnapshotVersion returns a version including the snapshot version: version and snapshot
func GetSnapshotVersion(version string) string {
	return newElasticVersion(version).SnapshotVersion
}

// GetVersion returns a version without snapshot or commit
func GetVersion(version string) string {
	return newElasticVersion(version).Version
}

// RemoveCommitFromSnapshot removes the commit from a version including commit and SNAPSHOT
func RemoveCommitFromSnapshot(s string) string {
	// regex = X.Y.Z-commit-SNAPSHOT
	re := regexp.MustCompile(`-\b[0-9a-f]{5,40}\b`)

	return re.ReplaceAllString(s, "")
}

// SnapshotHasCommit returns true if the snapshot version contains a commit format
func SnapshotHasCommit(s string) bool {
	// regex = X.Y.Z-commit-SNAPSHOT
	re := regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-\b[0-9a-f]{5,40}\b)(-SNAPSHOT)`)

	return re.MatchString(s)
}

// buildArtifactName builds the artifact name from the different coordinates for the artifact
func buildArtifactName(artifact string, artifactVersion string, OS string, arch string, extension string, isDocker bool) string {
	dockerString := ""
	if isDocker {
		dockerString = ".docker"
	}

	artifactVersion = GetSnapshotVersion(artifactVersion)

	lowerCaseExtension := strings.ToLower(extension)

	artifactName := fmt.Sprintf("%s-%s-%s-%s%s.%s", artifact, artifactVersion, OS, arch, dockerString, lowerCaseExtension)
	if lowerCaseExtension == "deb" || lowerCaseExtension == "rpm" {
		artifactName = fmt.Sprintf("%s-%s-%s%s.%s", artifact, artifactVersion, arch, dockerString, lowerCaseExtension)
	}

	beatsLocalPath := shell.GetEnv("BEATS_LOCAL_PATH", "")
	if beatsLocalPath != "" && isDocker {
		return fmt.Sprintf("%s-%s-%s-%s%s.%s", artifact, artifactVersion, OS, arch, dockerString, lowerCaseExtension)
	}

	useCISnapshots := shell.GetEnvBool("BEATS_USE_CI_SNAPSHOTS")
	// we detected that the docker name on CI is using a different structure
	// CI snapshots on GCP: elastic-agent-$VERSION-linux-$ARCH.docker.tar.gz
	// Elastic's snapshots: elastic-agent-$VERSION-docker-image-linux-$ARCH.tar.gz
	if !useCISnapshots && isDocker {
		dockerString = "docker-image"
		artifactName = fmt.Sprintf("%s-%s-%s-%s-%s.%s", artifact, artifactVersion, dockerString, OS, arch, lowerCaseExtension)
	}

	return artifactName
}

// fetchBeatsBinary it downloads the binary and returns the location of the downloaded file
// If the environment variable BEATS_LOCAL_PATH is set, then the artifact
// to be used will be defined by the local snapshot produced by the local build.
// Else, if the environment variable BEATS_USE_CI_SNAPSHOTS is set, then the artifact
// to be downloaded will be defined by the latest snapshot produced by the Beats CI.
func fetchBeatsBinary(ctx context.Context, artifactName string, artifact string, version string, timeoutFactor int, xpack bool) (string, error) {
	beatsLocalPath := shell.GetEnv("BEATS_LOCAL_PATH", "")
	if beatsLocalPath != "" {
		span, _ := apm.StartSpanOptions(ctx, "Fetching Beats binary", "beats.local.fetch-binary", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		defer span.End()

		distributions := path.Join(beatsLocalPath, artifact, "build", "distributions")
		if xpack {
			distributions = path.Join(beatsLocalPath, "x-pack", artifact, "build", "distributions")
		}

		log.Debugf("Using local snapshots for the %s: %s", artifact, distributions)

		fileNamePath, _ := filepath.Abs(path.Join(distributions, artifactName))
		_, err := os.Stat(fileNamePath)
		if err != nil || os.IsNotExist(err) {
			return fileNamePath, err
		}

		return fileNamePath, err
	}

	handleDownload := func(URL string) (string, error) {
		span, _ := apm.StartSpanOptions(ctx, "Fetching Beats binary", "beats.url.fetch-binary", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		defer span.End()

		if val, ok := binariesCache[URL]; ok {
			log.WithFields(log.Fields{
				"URL":  URL,
				"path": val,
			}).Debug("Retrieving binary from local cache")
			return val, nil
		}

		filePathFull, err := utils.DownloadFile(URL)
		if err != nil {
			return filePathFull, err
		}

		// use artifact name as file name to avoid having URL params in the name
		sanitizedFilePath := filepath.Join(path.Dir(filePathFull), artifactName)
		err = os.Rename(filePathFull, sanitizedFilePath)
		if err != nil {
			log.WithFields(log.Fields{
				"fileName":          filePathFull,
				"sanitizedFileName": sanitizedFilePath,
			}).Warn("Could not sanitize downloaded file name. Keeping old name")
			sanitizedFilePath = filePathFull
		}

		binariesCache[URL] = sanitizedFilePath

		return sanitizedFilePath, nil
	}

	var downloadURL string
	var err error

	useCISnapshots := shell.GetEnvBool("BEATS_USE_CI_SNAPSHOTS")
	if useCISnapshots {
		span, _ := apm.StartSpanOptions(ctx, "Fetching Beats binary", "beats.gcp.fetch-binary", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		defer span.End()

		log.Debugf("Using CI snapshots for %s", artifact)

		bucket, prefix, object := getGCPBucketCoordinates(artifactName, artifact)

		maxTimeout := time.Duration(timeoutFactor) * time.Minute

		downloadURL, err = utils.GetObjectURLFromBucket(bucket, prefix, object, maxTimeout)
		if err != nil {
			return "", err
		}

		return handleDownload(downloadURL)
	}

	downloadURL, err = GetElasticArtifactURL(artifactName, artifact, version)
	if err != nil {
		return "", err
	}

	return handleDownload(downloadURL)
}

// getGCPBucketCoordinates it calculates the bucket path in GCP
func getGCPBucketCoordinates(fileName string, artifact string) (string, string, string) {
	bucket := "beats-ci-artifacts"

	if strings.HasSuffix(artifact, "-ubi8") {
		artifact = strings.ReplaceAll(artifact, "-ubi8", "")
	}

	prefix := fmt.Sprintf("snapshots/%s", artifact)
	object := fileName

	// the commit SHA will identify univocally the artifact in the GCP storage bucket
	commitSHA := shell.GetEnv("GITHUB_CHECK_SHA1", "")
	if commitSHA != "" {
		log.WithFields(log.Fields{
			"commit": commitSHA,
			"file":   fileName,
		}).Debug("Using CI snapshots for a commit")
		prefix = fmt.Sprintf("commits/%s", commitSHA)
		object = artifact + "/" + fileName
	}

	return bucket, prefix, object
}
