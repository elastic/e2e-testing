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
	"strconv"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/cli/docker"
	curl "github.com/elastic/e2e-testing/cli/shell"
	log "github.com/sirupsen/logrus"
)

//nolint:unused
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

//nolint:unused
var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

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
// i.e. GetElasticArtifactVersion("7.x-SNAPSHOT")
// i.e. GetElasticArtifactVersion("pr-22000")
func GetElasticArtifactVersion(version string) string {
	if strings.HasPrefix(strings.ToLower(version), "pr-") {
		return version
	}

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
		log.WithFields(log.Fields{
			"error":   err,
			"version": version,
		}).Fatal("Failed to get version, aborting")
		return ""
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": version,
		}).Fatal("Could not parse the response body to retrieve the version, aborting")
		return ""
	}

	builds := jsonParsed.Path("version.builds")

	lastBuild := builds.Children()[0]
	latestVersion := lastBuild.Path("version").Data().(string)

	log.WithFields(log.Fields{
		"alias":   version,
		"version": latestVersion,
	}).Debug("Latest version for current version obtained")

	return latestVersion
}

// GetElasticArtifactURL returns the URL of a released artifact from two possible sources
// on the desired OS, architecture and file extension:
// 1. Observability CI Storage bucket
// 2. Elastic's artifact repository, building the JSON path query based
// i.e. GetElasticArtifactURL("elastic-agent", "7.x-SNAPSHOT", "linux", "x86_64", "tar.gz")
// i.e. GetElasticArtifactURL("elastic-agent", "7.x-SNAPSHOT", "x86_64", "rpm")
// i.e. GetElasticArtifactURL("elastic-agent", "7.x-SNAPSHOT", "amd64", "deb")
func GetElasticArtifactURL(artifact string, version string, operativeSystem string, arch string, extension string) (string, error) {
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
				"version":        version,
				"os":             operativeSystem,
				"arch":           arch,
				"extension":      extension,
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
			"artifact":  artifact,
			"version":   version,
			"os":        operativeSystem,
			"arch":      arch,
			"extension": extension,
		}).Error("Could not parse the response body for the artifact")
		return "", err
	}

	// elastic-agent-7.11.0-SNAPSHOT-linux-x86_64.tar.gz
	artifactPath := fmt.Sprintf("%s-%s-%s-%s.%s", artifact, version, operativeSystem, arch, extension)
	if extension == "deb" || extension == "rpm" {
		// elastic-agent-7.11.0-SNAPSHOT-x86_64.rpm
		// elastic-agent-7.11.0-SNAPSHOT-amd64.deb
		artifactPath = fmt.Sprintf("%s-%s-%s.%s", artifact, version, arch, extension)
	}

	packagesObject := jsonParsed.Path("packages")
	// we need to get keys with dots using Search instead of Path
	downloadObject := packagesObject.Search(artifactPath)
	downloadURL := downloadObject.Path("url").Data().(string)

	return downloadURL, nil
}

// GetObjectURLFromBucket extracts the media URL for the desired artifact from the
// Google Cloud Storage bucket used by the CI to push snapshots
func GetObjectURLFromBucket(bucket string, object string, maxtimeout time.Duration) (string, error) {
	exp := GetExponentialBackOff(maxtimeout)

	retryCount := 1

	currentPage := 0
	pageTokenQueryParam := ""
	mediaLink := ""

	storageAPI := func() error {
		r := curl.HTTPRequest{
			URL: fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o?prefix=pull-requests%s", bucket, pageTokenQueryParam),
		}

		response, err := curl.Get(r)
		if err != nil {
			log.WithFields(log.Fields{
				"bucket":      bucket,
				"elapsedTime": exp.GetElapsedTime(),
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
			"object":      object,
			"retries":     retryCount,
			"url":         r.URL,
		}).Trace("Google Cloud Storage API is available")

		jsonParsed, err := gabs.ParseJSON([]byte(response))
		if err != nil {
			log.WithFields(log.Fields{
				"bucket": bucket,
				"object": object,
			}).Warn("Could not parse the response body for the object")

			retryCount++

			return err
		}

		items := jsonParsed.Path("items").Children()

		log.WithFields(log.Fields{
			"bucket":      bucket,
			"elapsedTime": exp.GetElapsedTime(),
			"objects":     len(items),
			"retries":     retryCount,
		}).Debug("Objects found")

		for _, item := range items {
			itemID := item.Path("id").Data().(string)
			objectPath := bucket + "/" + object + "/"
			if strings.HasPrefix(itemID, objectPath) {
				mediaLink = item.Path("mediaLink").Data().(string)

				log.WithFields(log.Fields{
					"bucket":      bucket,
					"elapsedTime": exp.GetElapsedTime(),
					"medialink":   mediaLink,
					"object":      object,
					"retries":     retryCount,
				}).Debug("Media link found for the object")
				return nil
			}

			log.WithFields(log.Fields{
				"bucket":      bucket,
				"elapsedTime": exp.GetElapsedTime(),
				"object":      object,
				"itemID":      itemID,
				"retries":     retryCount,
			}).Trace("Media link not found")
		}

		if jsonParsed.Path("nextPageToken") == nil {
			log.WithFields(log.Fields{
				"currentPage": currentPage,
				"bucket":      bucket,
				"object":      object,
			}).Warn("Reached the end of the pages and the object was not found")

			return nil
		}

		nextPageToken := jsonParsed.Path("nextPageToken").Data().(string)
		pageTokenQueryParam = "&pageToken=" + nextPageToken
		currentPage++

		log.WithFields(log.Fields{
			"currentPage": currentPage,
			"bucket":      bucket,
			"elapsedTime": exp.GetElapsedTime(),
			"object":      object,
			"retries":     retryCount,
		}).Warn("Object not found in current page. Continuing")

		return fmt.Errorf("The %s object could not be found in the current page (%d) the %s bucket", object, currentPage, bucket)
	}

	err := backoff.Retry(storageAPI, exp)
	if err != nil {
		return "", err
	}
	if mediaLink == "" {
		return "", fmt.Errorf("Reached the end of the pages and the %s object was not found for the %s bucket", object, bucket)
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

// Sleep sleeps a number of seconds, including logs
func Sleep(seconds string) error {
	fields := log.Fields{
		"seconds": seconds,
	}

	s, err := strconv.Atoi(seconds)
	if err != nil {
		log.WithFields(fields).Errorf("Cannot convert %s to seconds", seconds)
		return err
	}

	log.WithFields(fields).Tracef("Waiting %s seconds", seconds)
	time.Sleep(time.Duration(s) * time.Second)

	return nil
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
