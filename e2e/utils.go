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
	shell "github.com/elastic/e2e-testing/cli/shell"
	log "github.com/sirupsen/logrus"
)

//nolint:unused
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

//nolint:unused
var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

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

// GetElasticArtifactURL returns the URL of a released artifact from two possible sources
// on the desired OS, architecture and file extension:
// 1. Observability CI Storage bucket
// 2. Elastic's artifact repository, building the JSON path query based
// i.e. GetElasticArtifactURL("elastic-agent", "8.0.0-SNAPSHOT", "linux", "x86_64", "tar.gz")
// If the environment variable ELASTIC_AGENT_DOWNLOAD_URL exists, then the artifact to be downloaded will
// be defined by that value
// Else, if the environment variable ELASTIC_AGENT_USE_CI_SNAPSHOTS is set, then the artifact
// to be downloaded will be defined by the latest snapshot produced by the Beats CI.
func GetElasticArtifactURL(artifact string, version string, OS string, arch string, extension string) (string, error) {
	downloadURL := os.Getenv("ELASTIC_AGENT_DOWNLOAD_URL")
	if downloadURL != "" {
		return downloadURL, nil
	}

	useCISnapshots, _ := shell.GetEnvBool("ELASTIC_AGENT_USE_CI_SNAPSHOTS")
	if useCISnapshots {
		// We will use the snapshots produced by Beats CI
		bucket := "beats-ci-artifacts"
		object := fmt.Sprintf("%s-%s-%s-%s.%s", artifact, version, OS, arch, extension)

		if agentVersion, exists := os.LookupEnv("ELASTIC_AGENT_VERSION"); exists {
			object = fmt.Sprintf("pull-requests/%s/%s-%s-%s-%s.%s", agentVersion, artifact, version, OS, arch, extension)
		}

		return GetObjectURLFromBucket(bucket, object)
	}

	exp := GetExponentialBackOff(time.Minute)

	retryCount := 1

	body := ""

	apiStatus := func() error {
		r := curl.HTTPRequest{
			URL: fmt.Sprintf("https://artifacts-api.elastic.co/v1/search/%s/%s", version, artifact),
		}

		response, err := curl.Get(r)
		if err != nil {
			log.WithFields(log.Fields{
				"artifact":       artifact,
				"version":        version,
				"os":             OS,
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
			"os":        OS,
			"arch":      arch,
			"extension": extension,
		}).Error("Could not parse the response body for the artifact")
		return "", err
	}

	artifactPath := fmt.Sprintf("%s-%s-%s-%s.%s", artifact, version, OS, arch, extension)
	packagesObject := jsonParsed.Path("packages")
	// we need to get keys with dots using Search instead of Path
	downloadObject := packagesObject.Search(artifactPath)
	downloadURL = downloadObject.Path("url").Data().(string)

	return downloadURL, nil
}

// GetObjectURLFromBucket extracts the media URL for the desired artifact from the
// Google Cloud Storage bucket used by the CI to push snapshots
func GetObjectURLFromBucket(bucket string, object string) (string, error) {
	exp := GetExponentialBackOff(time.Minute)

	retryCount := 1

	body := ""

	storageAPI := func() error {
		r := curl.HTTPRequest{
			URL: fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o", bucket),
		}

		response, err := curl.Get(r)
		if err != nil {
			log.WithFields(log.Fields{
				"bucket":         bucket,
				"elapsedTime":    exp.GetElapsedTime(),
				"error":          err,
				"object":         object,
				"retry":          retryCount,
				"statusEndpoint": r.URL,
			}).Warn("Google Cloud Storage API is not available yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"bucket":         bucket,
			"elapsedTime":    exp.GetElapsedTime(),
			"object":         object,
			"retries":        retryCount,
			"statusEndpoint": r.URL,
		}).Debug("Google Cloud Storage API is available")

		body = response
		return nil
	}

	err := backoff.Retry(storageAPI, exp)
	if err != nil {
		return "", err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"bucket": bucket,
			"object": object,
		}).Error("Could not parse the response body for the object")
		return "", err
	}

	for _, item := range jsonParsed.Path("items").Children() {
		itemID := item.Path("id").Data().(string)
		if strings.Contains(itemID, object) {
			return item.Path("mediaLink").Data().(string), nil
		}
	}

	return "", fmt.Errorf("The %s object could not be found in the %s bucket", object, bucket)
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
		}).Debug("File downloaded")

		fileReader = resp.Body

		return nil
	}

	log.WithFields(log.Fields{
		"url":  url,
		"path": filepath,
	}).Debug("Downloading file")

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

	log.WithFields(fields).Debugf("Waiting %s seconds", seconds)
	time.Sleep(time.Duration(s) * time.Second)

	return nil
}

// WaitForProcess polls a container executing "ps" command until the process is in the desired state (present or not),
// or a timeout happens
func WaitForProcess(host string, process string, desiredState string, maxTimeout time.Duration) error {
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
		}).Debug("Checking process desired state on the host")

		output, err := docker.ExecCommandIntoContainer(context.Background(), host, "root", []string{"pgrep", "-n", "-l", process})
		if err != nil {
			log.WithFields(log.Fields{
				"desiredState": desiredState,
				"elapsedTime":  exp.GetElapsedTime(),
				"error":        err,
				"host":         host,
				"process":      process,
				"retry":        retryCount,
			}).Warn("Could not execute 'pgrep -n -l' in the host")

			retryCount++

			return err
		}

		outputContainsProcess := strings.Contains(output, process)

		// both true or both false
		if mustBePresent == outputContainsProcess {
			log.WithFields(log.Fields{
				"desiredState":  desiredState,
				"host":          host,
				"mustBePresent": outputContainsProcess,
				"process":       process,
			}).Infof("Process desired state checked")

			return nil
		}

		if mustBePresent {
			err = fmt.Errorf("Process is not running in the host yet")
			log.WithFields(log.Fields{
				"desiredState": desiredState,
				"elapsedTime":  exp.GetElapsedTime(),
				"error":        err,
				"host":         host,
				"process":      process,
				"retry":        retryCount,
			}).Warn(err.Error())

			retryCount++

			return err
		}

		err = fmt.Errorf("Process is still running in the host")
		log.WithFields(log.Fields{
			"elapsedTime": exp.GetElapsedTime(),
			"error":       err,
			"host":        host,
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
