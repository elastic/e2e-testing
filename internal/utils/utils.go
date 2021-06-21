// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package utils

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	backoff "github.com/cenkalti/backoff/v4"
	curl "github.com/elastic/e2e-testing/internal/curl"
	internalio "github.com/elastic/e2e-testing/internal/io"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// to avoid fetching the same Elastic artifacts version, we are adding this map to cache the version of the Elastic artifacts,
// using as key the URL of the version. If another request is trying to fetch the same URL, it will return the string version
// of the already requested one.
var elasticVersionsCache = map[string]string{}

// to avoid downloading the same artifacts, we are adding this map to cache the URL of the downloaded binaries, using as key
// the URL of the artifact. If another installer is trying to download the same URL, it will return the location of the
// already downloaded artifact.
var binariesCache = map[string]string{}

//nolint:unused
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

//nolint:unused
var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

// buildArtifactName builds the artifact name from the different coordinates for the artifact
func buildArtifactName(artifact string, artifactVersion string, OS string, arch string, extension string, isDocker bool) string {
	dockerString := ""
	if isDocker {
		dockerString = ".docker"
	}

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
	binaryPath, err := fetchBeatsBinary(ctx, binaryName, artifact, version, TimeoutFactor, xpack)
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

		filePath, err := DownloadFile(URL)
		if err != nil {
			return filePath, err
		}

		// use artifact name as file name to avoid having URL params in the name
		sanitizedFilePath := path.Join(path.Dir(filePath), artifactName)
		err = os.Rename(filePath, sanitizedFilePath)
		if err != nil {
			log.WithFields(log.Fields{
				"fileName":          filePath,
				"sanitizedFileName": sanitizedFilePath,
			}).Warn("Could not sanitize downloaded file name. Keeping old name")
			sanitizedFilePath = filePath
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

		bucket, prefix, object := getGCPBucketCoordinates(artifactName, artifact, version)

		maxTimeout := time.Duration(timeoutFactor) * time.Minute

		downloadURL, err = GetObjectURLFromBucket(bucket, prefix, object, maxTimeout)
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

// GetArchitecture retrieves if the underlying system platform is arm64 or amd64
func GetArchitecture() string {
	arch := "amd64"

	envArch := os.Getenv("GOARCH")
	if envArch == "arm64" {
		arch = "arm64"
	}

	log.Debugf("Golang's architecture is %s (%s)", arch, envArch)
	return arch
}

// getGCPBucketCoordinates it calculates the bucket path in GCP
func getGCPBucketCoordinates(fileName string, artifact string, version string) (string, string, string) {
	bucket := "beats-ci-artifacts"
	prefix := fmt.Sprintf("snapshots/%s", artifact)
	object := fileName

	// the commit SHA will identify univocally the artifact in the GCP storage bucket
	commitSHA := shell.GetEnv("GITHUB_CHECK_SHA1", "")
	if commitSHA != "" {
		log.WithFields(log.Fields{
			"commit":  commitSHA,
			"version": version,
		}).Debug("Using CI snapshots for a commit")
		prefix = fmt.Sprintf("commits/%s", commitSHA)
		object = artifact + "/" + fileName
	}

	return bucket, prefix, object
}

// GetElasticArtifactVersion returns the current version:
// 1. Elastic's artifact repository, building the JSON path query based
// If the version is a PR, then it will return the version without checking the artifacts API
// i.e. GetElasticArtifactVersion("$VERSION")
func GetElasticArtifactVersion(version string) (string, error) {
	cacheKey := fmt.Sprintf("https://artifacts-api.elastic.co/v1/versions/%s/?x-elastic-no-kpi=true", version)

	if val, ok := elasticVersionsCache[cacheKey]; ok {
		log.WithFields(log.Fields{
			"URL":     cacheKey,
			"version": val,
		}).Debug("Retrieving version from local cache")
		return val, nil
	}

	exp := GetExponentialBackOff(time.Minute)

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

// GetElasticArtifactURL returns the URL of a released artifact, which its full name is defined in the first argument,
// from Elastic's artifact repository, building the JSON path query based on the full name
// i.e. GetElasticArtifactURL("elastic-agent-$VERSION-$ARCH.deb", "elastic-agent", "$VERSION")
// i.e. GetElasticArtifactURL("elastic-agent-$VERSION-x86_64.rpm", "elastic-agent","$VERSION")
// i.e. GetElasticArtifactURL("elastic-agent-$VERSION-linux-$ARCH.tar.gz", "elastic-agent","$VERSION")
func GetElasticArtifactURL(artifactName string, artifact string, version string) (string, error) {
	exp := GetExponentialBackOff(time.Minute)

	retryCount := 1

	body := ""

	apiStatus := func() error {
		r := curl.HTTPRequest{
			URL: fmt.Sprintf("https://artifacts-api.elastic.co/v1/search/%s/%s?x-elastic-no-kpi=true", version, artifact),
		}

		response, err := curl.Get(r)
		if err != nil {
			log.WithFields(log.Fields{
				"artifact":       artifact,
				"artifactName":   artifactName,
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
			"artifact":     artifact,
			"artifactName": artifactName,
			"version":      version,
		}).Error("Could not parse the response body for the artifact")
		return "", err
	}

	log.WithFields(log.Fields{
		"retries":      retryCount,
		"artifact":     artifact,
		"artifactName": artifactName,
		"elapsedTime":  exp.GetElapsedTime(),
		"version":      version,
	}).Trace("Artifact found")

	packagesObject := jsonParsed.Path("packages")
	// we need to get keys with dots using Search instead of Path
	downloadObject := packagesObject.Search(artifactName)
	downloadURL := downloadObject.Path("url").Data().(string)

	return downloadURL, nil
}

// GetObjectURLFromBucket extracts the media URL for the desired artifact from the
// Google Cloud Storage bucket used by the CI to push snapshots
func GetObjectURLFromBucket(bucket string, prefix string, object string, maxtimeout time.Duration) (string, error) {
	exp := GetExponentialBackOff(maxtimeout)

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

		return fmt.Errorf("The %s object could not be found in the current page (%d) the %s bucket and %s prefix", object, currentPage, bucket, prefix)
	}

	err := backoff.Retry(storageAPI, exp)
	if err != nil {
		return "", err
	}
	if mediaLink == "" {
		return "", fmt.Errorf("Reached the end of the pages and the %s object was not found for the %s bucket and %s prefix", object, bucket, prefix)
	}

	return mediaLink, nil
}

// DownloadFile will download a url and store it in a temporary path.
// It writes to the destination file as it downloads it, without
// loading the entire file into memory.
func DownloadFile(url string) (string, error) {
	tempParentDir := path.Join(os.TempDir(), uuid.NewString())
	internalio.MkdirAll(tempParentDir)

	tempFile, err := os.Create(path.Join(tempParentDir, path.Base(url)))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"url":   url,
		}).Error("Error creating file")
		return "", err
	}
	defer tempFile.Close()

	filepath := tempFile.Name()

	exp := GetExponentialBackOff(3)

	retryCount := 1
	var fileReader io.ReadCloser

	download := func() error {
		resp, err := http.Get(url)
		if err != nil {
			log.WithFields(log.Fields{
				"elapsedTime": exp.GetElapsedTime(),
				"error":       err,
				"path":        filepath,
				"retry":       retryCount,
				"url":         url,
			}).Warn("Could not download the file")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"elapsedTime": exp.GetElapsedTime(),
			"retries":     retryCount,
			"path":        filepath,
			"url":         url,
		}).Trace("File downloaded")

		fileReader = resp.Body

		return nil
	}

	log.WithFields(log.Fields{
		"url":  url,
		"path": filepath,
	}).Trace("Downloading file")

	err = backoff.Retry(download, exp)
	if err != nil {
		return "", err
	}
	defer fileReader.Close()

	_, err = io.Copy(tempFile, fileReader)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"url":   url,
			"path":  filepath,
		}).Error("Could not write file")

		return filepath, err
	}

	_ = os.Chmod(tempFile.Name(), 0666)

	return filepath, nil
}

func getBucketSearchNextPageParam(jsonParsed *gabs.Container) string {
	token := jsonParsed.Path("nextPageToken")
	if token == nil {
		return ""
	}

	nextPageToken := token.Data().(string)
	return "&pageToken=" + nextPageToken
}

// IsCommit returns true if the string matches commit format
func IsCommit(s string) bool {
	re := regexp.MustCompile(`\b[0-9a-f]{5,40}\b`)

	return re.MatchString(s)
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

	return "", fmt.Errorf("The %s object could not be found in the current page (%d) in the %s bucket and %s prefix", object, currentPage, bucket, prefix)
}

//nolint:unused
func randomStringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// RandomString generates a random string with certain length
func RandomString(length int) string {
	return randomStringWithCharset(length, charset)
}

// Sleep sleeps a duration, including logs
func Sleep(duration time.Duration) error {
	fields := log.Fields{
		"duration": duration,
	}

	log.WithFields(fields).Tracef("Waiting %v", duration)
	time.Sleep(duration)

	return nil
}

// GetDockerNamespaceEnvVar returns the Docker namespace whether we use one of the CI snapshots or
// the images produced by local Beats build, or not.
// If an error occurred reading the environment, will return the passed namespace as fallback
func GetDockerNamespaceEnvVar(fallback string) string {
	beatsLocalPath := shell.GetEnv("BEATS_LOCAL_PATH", "")
	useCISnapshots := shell.GetEnvBool("BEATS_USE_CI_SNAPSHOTS")
	if useCISnapshots || beatsLocalPath != "" {
		return "observability-ci"
	}
	return fallback
}
