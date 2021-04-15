// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package e2e

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
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
	"github.com/elastic/e2e-testing/cli/docker"
	"github.com/elastic/e2e-testing/cli/shell"
	curl "github.com/elastic/e2e-testing/cli/shell"
	log "github.com/sirupsen/logrus"
)

// to avoid downloading the same artifacts, we are adding this map to cache the URL of the downloaded binaries, using as key
// the URL of the artifact. If another installer is trying to download the same URL, it will return the location of the
// already downloaded artifact.
var binariesCache = map[string]string{}

//nolint:unused
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

//nolint:unused
var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

// BuildArtifactName builds the artifact name from the different coordinates for the artifact
func BuildArtifactName(artifact string, version string, fallbackVersion string, OS string, arch string, extension string, isDocker bool) string {
	dockerString := ""
	if isDocker {
		dockerString = ".docker"
	}

	artifactVersion := CheckPRVersion(version, fallbackVersion)

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
	// CI snapshots on GCP: elastic-agent-$VERSION-linux-amd64.docker.tar.gz
	// Elastic's snapshots: elastic-agent-$VERSION-docker-image-linux-amd64.tar.gz
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

// FetchBeatsBinary it downloads the binary and returns the location of the downloaded file
// If the environment variable BEATS_LOCAL_PATH is set, then the artifact
// to be used will be defined by the local snapshot produced by the local build.
// Else, if the environment variable BEATS_USE_CI_SNAPSHOTS is set, then the artifact
// to be downloaded will be defined by the latest snapshot produced by the Beats CI.
func FetchBeatsBinary(artifactName string, artifact string, version string, fallbackVersion string, timeoutFactor int, xpack bool) (string, error) {
	beatsLocalPath := shell.GetEnv("BEATS_LOCAL_PATH", "")
	if beatsLocalPath != "" {
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

		binariesCache[URL] = filePath

		return filePath, nil
	}

	var downloadURL string
	var err error

	useCISnapshots := shell.GetEnvBool("BEATS_USE_CI_SNAPSHOTS")
	if useCISnapshots {
		log.Debugf("Using CI snapshots for %s", artifact)

		bucket, prefix, object := getGCPBucketCoordinates(artifactName, artifact, version, fallbackVersion)

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

// getGCPBucketCoordinates it calculates the bucket path in GCP
func getGCPBucketCoordinates(fileName string, artifact string, version string, fallbackVersion string) (string, string, string) {
	bucket := "beats-ci-artifacts"
	prefix := fmt.Sprintf("snapshots/%s", artifact)
	object := fileName

	// the commit SHA will identify univocally the artifact in the GCP storage bucket
	commitSHA := shell.GetEnv("GITHUB_CHECK_SHA1", "")
	if commitSHA != "" {
		log.WithFields(log.Fields{
			"commit":  commitSHA,
			"PR":      version,
			"version": fallbackVersion,
		}).Debug("Using CI snapshots for a commit")
		prefix = fmt.Sprintf("commits/%s", commitSHA)
		object = artifact + "/" + fileName
	}

	return bucket, prefix, object
}

// GetExponentialBackOff returns a preconfigured exponential backoff instance
func GetExponentialBackOff(elapsedTime time.Duration) *backoff.ExponentialBackOff {
	var (
		initialInterval     = 500 * time.Millisecond
		randomizationFactor = 0.5
		multiplier          = 2.0
		maxInterval         = 5 * time.Second
		maxElapsedTime      = elapsedTime
	)

	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = initialInterval
	exp.RandomizationFactor = randomizationFactor
	exp.Multiplier = multiplier
	exp.MaxInterval = maxInterval
	exp.MaxElapsedTime = maxElapsedTime

	return exp
}

// GetElasticArtifactVersion returns the current version:
// 1. Elastic's artifact repository, building the JSON path query based
// If the version is a PR, then it will return the version without checking the artifacts API
// i.e. GetElasticArtifactVersion("$VERSION")
func GetElasticArtifactVersion(version string) (string, error) {
	exp := GetExponentialBackOff(time.Minute)

	retryCount := 1

	body := ""

	apiStatus := func() error {
		r := curl.HTTPRequest{
			URL: fmt.Sprintf("https://artifacts-api.elastic.co/v1/versions/%s/?x-elastic-no-kpi=true", version),
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

	return latestVersion, nil
}

// GetElasticArtifactURL returns the URL of a released artifact, which its full name is defined in the first argument,
// from Elastic's artifact repository, building the JSON path query based on the full name
// i.e. GetElasticArtifactURL("elastic-agent-$VERSION-amd64.deb", "elastic-agent", "$VERSION")
// i.e. GetElasticArtifactURL("elastic-agent-$VERSION-x86_64.rpm", "elastic-agent","$VERSION")
// i.e. GetElasticArtifactURL("elastic-agent-$VERSION-linux-amd64.tar.gz", "elastic-agent","$VERSION")
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
	tempFile, err := ioutil.TempFile(os.TempDir(), path.Base(url))
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

// WaitForProcess polls a container executing "ps" command until the process is in the desired state (present or not),
// or a timeout happens
func WaitForProcess(containerName string, process string, desiredState string, maxTimeout time.Duration) error {
	exp := GetExponentialBackOff(maxTimeout)

	mustBePresent := false
	if desiredState == "started" {
		mustBePresent = true
	}
	retryCount := 1

	processStatus := func() error {
		log.WithFields(log.Fields{
			"desiredState": desiredState,
			"process":      process,
		}).Trace("Checking process desired state on the container")

		output, err := docker.ExecCommandIntoContainer(context.Background(), containerName, "root", []string{"pgrep", "-n", "-l", "-f", process})
		if err != nil {
			log.WithFields(log.Fields{
				"desiredState":  desiredState,
				"elapsedTime":   exp.GetElapsedTime(),
				"error":         err,
				"container":     containerName,
				"mustBePresent": mustBePresent,
				"process":       process,
				"retry":         retryCount,
			}).Warn("Could not execute 'pgrep -n -l -f' in the container")

			retryCount++

			return err
		}

		outputContainsProcess := strings.Contains(output, process)

		// both true or both false
		if mustBePresent == outputContainsProcess {
			log.WithFields(log.Fields{
				"desiredState":  desiredState,
				"container":     containerName,
				"mustBePresent": mustBePresent,
				"process":       process,
			}).Infof("Process desired state checked")

			return nil
		}

		if mustBePresent {
			err = fmt.Errorf("%s process is not running in the container yet", process)
			log.WithFields(log.Fields{
				"desiredState": desiredState,
				"elapsedTime":  exp.GetElapsedTime(),
				"error":        err,
				"container":    containerName,
				"process":      process,
				"retry":        retryCount,
			}).Warn(err.Error())

			retryCount++

			return err
		}

		err = fmt.Errorf("%s process is still running in the container", process)
		log.WithFields(log.Fields{
			"elapsedTime": exp.GetElapsedTime(),
			"error":       err,
			"container":   containerName,
			"process":     process,
			"state":       desiredState,
			"retry":       retryCount,
		}).Warn(err.Error())

		retryCount++

		return err
	}

	err := backoff.Retry(processStatus, exp)
	if err != nil {
		return err
	}

	return nil
}
