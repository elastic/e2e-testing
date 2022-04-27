// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package downloads

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

// BeatsLocalPath is the path to a local copy of the Beats git repository
// It can be overriden by BEATS_LOCAL_PATH env var. Using the empty string as a default.
var BeatsLocalPath = ""

// to avoid downloading the same artifacts, we are adding this map to cache the URL of the downloaded binaries, using as key
// the URL of the artifact. If another installer is trying to download the same URL, it will return the location of the
// already downloaded artifact.
var binariesCache = map[string]string{}

// to avoid fetching the same Elastic artifacts version, we are adding this map to cache the version of the Elastic artifacts,
// using as key the URL of the version. If another request is trying to fetch the same URL, it will return the string version
// of the already requested one.
var elasticVersionsCache = map[string]string{}

// GithubCommitSha1 represents the value of the "GITHUB_CHECK_SHA1" environment variable
var GithubCommitSha1 string

// GithubRepository represents the value of the "GITHUB_CHECK_REPO" environment variable
// Default is "elastic-agent"
var GithubRepository string

func init() {
	GithubCommitSha1 = shell.GetEnv("GITHUB_CHECK_SHA1", "")
	GithubRepository = shell.GetEnv("GITHUB_CHECK_REPO", "elastic-agent")

	BeatsLocalPath = shell.GetEnv("BEATS_LOCAL_PATH", BeatsLocalPath)
	if BeatsLocalPath != "" {
		log.Infof(`Beats local path will be used for artifacts. Please make sure all binaries are properly built in their "build/distributions" folder: %s`, BeatsLocalPath)
	}
}

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
	if GithubCommitSha1 != "" {
		return fallbackVersion
	}

	return version
}

// FetchElasticArtifact fetches an artifact from the right repository, returning binary name, path and error
func FetchElasticArtifact(ctx context.Context, artifact string, version string, os string, arch string, extension string, isDocker bool, xpack bool) (string, string, error) {
	useCISnapshots := GithubCommitSha1 != ""

	return FetchElasticArtifactForSnapshots(ctx, useCISnapshots, artifact, version, os, arch, extension, isDocker, xpack)
}

// FetchElasticArtifactForSnapshots fetches an artifact from the right repository, returning binary name, path and error
func FetchElasticArtifactForSnapshots(ctx context.Context, useCISnapshots bool, artifact string, version string, os string, arch string, extension string, isDocker bool, xpack bool) (string, string, error) {
	binaryName := buildArtifactName(artifact, version, os, arch, extension, isDocker)
	binaryPath, err := FetchProjectBinaryForSnapshots(ctx, useCISnapshots, artifact, binaryName, artifact, version, utils.TimeoutFactor, xpack, "", false)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":  artifact,
			"version":   version,
			"os":        os,
			"arch":      arch,
			"extension": extension,
			"error":     err,
		}).Error("Could not download the binary for the Elastic artifact")
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
// It also returns the URL of the sha512 file of the released artifact.
// i.e. GetElasticArtifactURL("elastic-agent-$VERSION-$ARCH.deb", "elastic-agent", "$VERSION")
// i.e. GetElasticArtifactURL("elastic-agent-$VERSION-x86_64.rpm", "elastic-agent","$VERSION")
// i.e. GetElasticArtifactURL("elastic-agent-$VERSION-linux-$ARCH.tar.gz", "elastic-agent","$VERSION")
func GetElasticArtifactURL(artifactName string, artifact string, version string) (string, string, error) {
	resolver := NewArtifactURLResolver(artifactName, artifact, version)

	return resolver.Resolve()
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

// UseBeatsCISnapshots check if CI snapshots should be used for the Beats, where the given SHA commit
// lives in the beats repository
func UseBeatsCISnapshots() bool {
	return useCISnapshots("beats")
}

// UseElasticAgentCISnapshots check if CI snapshots should be used for the Elastic Agent, where the given SHA commit
// lives in the elastic-agent repository
func UseElasticAgentCISnapshots() bool {
	return useCISnapshots("elastic-agent")
}

// useCISnapshots check if CI snapshots should be used, passing a function that evaluates the repository in which
// the given Sha commit has context. I.e. a commit in the elastic-agent repository should pass a function that
func useCISnapshots(repository string) bool {
	if GithubCommitSha1 != "" && strings.EqualFold(GithubRepository, repository) {
		return true
	}
	return false
}

// buildArtifactName builds the artifact name from the different coordinates for the artifact
func buildArtifactName(artifact string, artifactVersion string, OS string, arch string, extension string, isDocker bool) string {
	dockerString := ""

	lowerCaseExtension := strings.ToLower(extension)
	artifactVersion = GetSnapshotVersion(artifactVersion)

	useCISnapshots := GithubCommitSha1 != ""

	if isDocker {
		// we detected that the docker name on CI is using a different structure
		// CI snapshots on GCP: elastic-agent-$VERSION-linux-$ARCH.docker.tar.gz
		// Elastic's snapshots: elastic-agent-$VERSION-docker-image-linux-$ARCH.tar.gz
		dockerString = ".docker"
		if !useCISnapshots {
			dockerString = "-docker-image"
		}
	}

	if BeatsLocalPath != "" && isDocker {
		dockerString = ".docker"
		return fmt.Sprintf("%s-%s-%s-%s%s.%s", artifact, artifactVersion, OS, arch, dockerString, lowerCaseExtension)
	}

	if !useCISnapshots && isDocker {
		return fmt.Sprintf("%s-%s%s-%s-%s.%s", artifact, artifactVersion, dockerString, OS, arch, lowerCaseExtension)
	}

	if lowerCaseExtension == "deb" || lowerCaseExtension == "rpm" {
		return fmt.Sprintf("%s-%s-%s%s.%s", artifact, artifactVersion, arch, dockerString, lowerCaseExtension)
	}

	return fmt.Sprintf("%s-%s-%s-%s%s.%s", artifact, artifactVersion, OS, arch, dockerString, lowerCaseExtension)

}

// FetchBeatsBinary it downloads the binary and returns the location of the downloaded file
// If the environment variable BEATS_LOCAL_PATH is set, then the artifact
// to be used will be defined by the local snapshot produced by the local build.
// Else, if the environment variable GITHUB_CHECK_SHA1 is set, then the artifact
// to be downloaded will be defined by the snapshot produced by the Beats CI for that commit.
func FetchBeatsBinary(ctx context.Context, artifactName string, artifact string, version string, timeoutFactor int, xpack bool, downloadPath string, downloadSHAFile bool) (string, error) {
	return FetchProjectBinary(ctx, "beats", artifactName, artifact, version, timeoutFactor, xpack, downloadPath, downloadSHAFile)
}

// FetchProjectBinary it downloads the binary and returns the location of the downloaded file
// If the environment variable BEATS_LOCAL_PATH is set, then the artifact
// to be used will be defined by the local snapshot produced by the local build.
// Else, if the environment variable GITHUB_CHECK_SHA1 is set, then the artifact
// to be downloaded will be defined by the snapshot produced by the Beats CI for that commit.
func FetchProjectBinary(ctx context.Context, project string, artifactName string, artifact string, version string, timeoutFactor int, xpack bool, downloadPath string, downloadSHAFile bool) (string, error) {
	useCISnapshots := GithubCommitSha1 != ""

	return FetchProjectBinaryForSnapshots(ctx, useCISnapshots, project, artifactName, artifact, version, timeoutFactor, xpack, downloadPath, downloadSHAFile)
}

// FetchProjectBinaryForSnapshots it downloads the binary and returns the location of the downloaded file
// If the environment variable BEATS_LOCAL_PATH is set, then the artifact
// to be used will be defined by the local snapshot produced by the local build.
// Else, if the useCISnapshots argument is set to true, then the artifact
// to be downloaded will be defined by the snapshot produced by the Beats CI or Fleet CI for that commit.
func FetchProjectBinaryForSnapshots(ctx context.Context, useCISnapshots bool, project string, artifactName string, artifact string, version string, timeoutFactor int, xpack bool, downloadPath string, downloadSHAFile bool) (string, error) {
	if BeatsLocalPath != "" && project == "beats" {
		span, _ := apm.StartSpanOptions(ctx, "Fetching Project binary", "project.local.fetch-binary", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		span.Context.SetLabel("project", project)
		defer span.End()

		distributions := path.Join(BeatsLocalPath, artifact, "build", "distributions")
		if xpack {
			distributions = path.Join(BeatsLocalPath, "x-pack", artifact, "build", "distributions")
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
		name := artifactName
		downloadRequest := utils.DownloadRequest{
			DownloadPath: downloadPath,
			URL:          URL,
		}
		span, _ := apm.StartSpanOptions(ctx, "Fetching Project binary", "project.url.fetch-binary", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		span.Context.SetLabel("project", project)
		defer span.End()

		if val, ok := binariesCache[URL]; ok {
			log.WithFields(log.Fields{
				"URL":  URL,
				"path": val,
			}).Debug("Retrieving binary from local cache")
			return val, nil
		}

		err := utils.DownloadFile(&downloadRequest)
		if err != nil {
			return downloadRequest.UnsanitizedFilePath, err
		}

		if strings.HasSuffix(URL, ".sha512") {
			name = fmt.Sprintf("%s.sha512", name)
		}
		// use artifact name as file name to avoid having URL params in the name
		sanitizedFilePath := filepath.Join(path.Dir(downloadRequest.UnsanitizedFilePath), name)
		err = os.Rename(downloadRequest.UnsanitizedFilePath, sanitizedFilePath)
		if err != nil {
			log.WithFields(log.Fields{
				"fileName":          downloadRequest.UnsanitizedFilePath,
				"sanitizedFileName": sanitizedFilePath,
			}).Warn("Could not sanitize downloaded file name. Keeping old name")
			sanitizedFilePath = downloadRequest.UnsanitizedFilePath
		}

		binariesCache[URL] = sanitizedFilePath

		return sanitizedFilePath, nil
	}

	var downloadURL, downloadShaURL string
	var err error

	if useCISnapshots {
		span, _ := apm.StartSpanOptions(ctx, "Fetching Beats binary", "beats.gcp.fetch-binary", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		defer span.End()

		log.Debugf("Using CI snapshots for %s", artifact)

		maxTimeout := time.Duration(timeoutFactor) * time.Minute

		variant := ""
		if strings.HasSuffix(artifact, "-ubi8") {
			variant = "ubi8"
		}

		// look up the bucket in this particular order:
		// 1. the project layout (elastic-agent, fleet-server)
		// 2. the new beats layout (beats)
		// 3. the legacy Beats layout (commits/snapshots)
		resolvers := []BucketURLResolver{
			NewProjectURLResolver(FleetCIArtifactsBase, project, artifactName, variant),
			NewProjectURLResolver(BeatsCIArtifactsBase, project, artifactName, variant),
			NewBeatsURLResolver(artifact, artifactName, variant),
			NewBeatsLegacyURLResolver(artifact, artifactName, variant),
		}

		downloadURL, err = getObjectURLFromResolvers(resolvers, maxTimeout)
		if err != nil {
			return "", err
		}
		downloadLocation, err := handleDownload(downloadURL)

		// check if sha file should be downloaded, else return
		if !downloadSHAFile {
			return downloadLocation, err
		}

		sha512ArtifactName := fmt.Sprintf("%s.sha512", artifactName)

		sha512Resolvers := []BucketURLResolver{
			NewProjectURLResolver(FleetCIArtifactsBase, project, sha512ArtifactName, variant),
			NewProjectURLResolver(BeatsCIArtifactsBase, project, sha512ArtifactName, variant),
			NewBeatsURLResolver(artifact, sha512ArtifactName, variant),
			NewBeatsLegacyURLResolver(artifact, sha512ArtifactName, variant),
		}

		downloadURL, err = getObjectURLFromResolvers(sha512Resolvers, maxTimeout)
		if err != nil {
			return "", err
		}
		return handleDownload(downloadURL)
	}

	elasticAgentNamespace := project
	if strings.EqualFold(elasticAgentNamespace, "elastic-agent") {
		elasticAgentNamespace = "beats"
	}

	// look up the binaries, first checking releases, then artifacts
	downloadURLResolvers := []DownloadURLResolver{
		NewReleaseURLResolver(elasticAgentNamespace, artifactName, artifact),
		NewArtifactURLResolver(artifactName, artifact, version),
	}
	downloadURL, downloadShaURL, err = getDownloadURLFromResolvers(downloadURLResolvers)
	if err != nil {
		return "", err
	}
	downloadLocation, err := handleDownload(downloadURL)
	if err != nil {
		return "", err
	}
	if downloadSHAFile && downloadShaURL != "" {
		downloadLocation, err = handleDownload(downloadShaURL)
	}
	return downloadLocation, err
}

func getBucketSearchNextPageParam(jsonParsed *gabs.Container) string {
	token := jsonParsed.Path("nextPageToken")
	if token == nil {
		return ""
	}

	nextPageToken := token.Data().(string)
	return "&pageToken=" + nextPageToken
}

// getDownloadURLFromResolvers returns the URL for the desired artifacts
func getDownloadURLFromResolvers(resolvers []DownloadURLResolver) (string, string, error) {
	for i, resolver := range resolvers {
		url, shaURL, err := resolver.Resolve()
		if err != nil {
			if i < len(resolvers)-1 {
				log.WithFields(log.Fields{
					"resolver": resolver,
				}).Warn("Object not found. Trying with another download resolver")
				continue
			} else {
				log.Error("Object not found. There is no other download resolver")
				return "", "", err
			}
		}

		return url, shaURL, nil
	}

	return "", "", fmt.Errorf("the artifact was not found")
}

// getObjectURLFromResolvers extracts the media URL for the desired artifact from the
// Google Cloud Storage bucket used by the CI to push snapshots
func getObjectURLFromResolvers(resolvers []BucketURLResolver, maxtimeout time.Duration) (string, error) {
	for i, resolver := range resolvers {
		bucket, prefix, object := resolver.Resolve()

		downloadURL, err := getObjectURLFromBucket(bucket, prefix, object, maxtimeout)
		if err != nil {
			if i < len(resolvers)-1 {
				log.WithFields(log.Fields{
					"resolver": resolver,
				}).Warn("Object not found. Trying with another artifact resolver")
				continue
			} else {
				log.Error("Object not found. There is no other artifact resolver")
				return "", err
			}
		}

		return downloadURL, nil
	}

	return "", fmt.Errorf("the artifact was not found")
}

// getObjectURLFromBucket extracts the media URL for the desired artifact from the
// Google Cloud Storage bucket used by the CI to push snapshots
func getObjectURLFromBucket(bucket string, prefix string, object string, maxtimeout time.Duration) (string, error) {
	exp := utils.GetExponentialBackOff(maxtimeout)

	retryCount := 1

	currentPage := 0
	pageTokenQueryParam := ""
	mediaLink := ""

	storageAPI := func() error {
		r := curl.HTTPRequest{
			URL: fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o?prefix=%s%s", bucket, prefix, pageTokenQueryParam),
		}

		response, err := curl.Get(r)
		if err != nil {
			log.WithFields(log.Fields{
				"bucket":      bucket,
				"elapsedTime": exp.GetElapsedTime(),
				"prefix":      prefix,
				"error":       err,
				"object":      object,
				"retry":       retryCount,
			}).Warn("Google Cloud Storage API is not available yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"bucket":      bucket,
			"elapsedTime": exp.GetElapsedTime(),
			"prefix":      prefix,
			"object":      object,
			"retries":     retryCount,
			"url":         r.URL,
		}).Trace("Google Cloud Storage API is available")

		jsonParsed, err := gabs.ParseJSON([]byte(response))
		if err != nil {
			log.WithFields(log.Fields{
				"bucket": bucket,
				"prefix": prefix,
				"object": object,
			}).Warn("Could not parse the response body for the object")

			retryCount++

			return err
		}

		mediaLink, err = processBucketSearchPage(jsonParsed, currentPage, bucket, prefix, object)
		if err != nil {
			log.WithFields(log.Fields{
				"currentPage": currentPage,
				"bucket":      bucket,
				"prefix":      prefix,
				"object":      object,
			}).Warn(err.Error())
		} else if mediaLink != "" {
			log.WithFields(log.Fields{
				"bucket":      bucket,
				"elapsedTime": exp.GetElapsedTime(),
				"prefix":      prefix,
				"medialink":   mediaLink,
				"object":      object,
				"retries":     retryCount,
			}).Debug("Media link found for the object")
			return nil
		}

		pageTokenQueryParam = getBucketSearchNextPageParam(jsonParsed)
		if pageTokenQueryParam == "" {
			log.WithFields(log.Fields{
				"currentPage": currentPage,
				"bucket":      bucket,
				"prefix":      prefix,
				"object":      object,
			}).Warn("Reached the end of the pages and the object was not found")

			return nil
		}

		currentPage++

		log.WithFields(log.Fields{
			"currentPage": currentPage,
			"bucket":      bucket,
			"elapsedTime": exp.GetElapsedTime(),
			"prefix":      prefix,
			"object":      object,
			"retries":     retryCount,
		}).Warn("Object not found in current page. Continuing")

		return fmt.Errorf("the %s object could not be found in the current page (%d) the %s bucket and %s prefix", object, currentPage, bucket, prefix)
	}

	err := backoff.Retry(storageAPI, exp)
	if err != nil {
		return "", err
	}
	if mediaLink == "" {
		return "", fmt.Errorf("reached the end of the pages and the %s object was not found for the %s bucket and %s prefix", object, bucket, prefix)
	}

	return mediaLink, nil
}

func processBucketSearchPage(jsonParsed *gabs.Container, currentPage int, bucket string, prefix string, object string) (string, error) {
	items := jsonParsed.Path("items").Children()

	log.WithFields(log.Fields{
		"bucket":  bucket,
		"prefix":  prefix,
		"objects": len(items),
		"object":  object,
	}).Debug("Objects found")

	for _, item := range items {
		itemID := item.Path("id").Data().(string)
		objectPath := bucket + "/" + prefix + "/" + object + "/"
		if strings.HasPrefix(itemID, objectPath) {
			mediaLink := item.Path("mediaLink").Data().(string)

			log.Infof("medialink: %s", mediaLink)
			return mediaLink, nil
		}
	}

	return "", fmt.Errorf("the %s object could not be found in the current page (%d) in the %s bucket and %s prefix", object, currentPage, bucket, prefix)
}
