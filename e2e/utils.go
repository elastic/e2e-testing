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

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/cli/docker"
	log "github.com/sirupsen/logrus"
)

//nolint:unused
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

//nolint:unused
var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// GetEnv returns an environment variable as string
func GetEnv(envVar string, defaultValue string) string {
	if value, exists := os.LookupEnv(envVar); exists {
		return value
	}

	return defaultValue
}

// GetIntegerFromEnv returns an environment variable as integer, including a default value
func GetIntegerFromEnv(envVar string, defaultValue int) int {
	if value, exists := os.LookupEnv(envVar); exists {
		v, err := strconv.Atoi(value)
		if err == nil {
			return v
		}
	}

	return defaultValue
}

// getExponentialBackOff returns a preconfigured exponential backoff instance
//nolint:unused
func getExponentialBackOff(elapsedTime time.Duration) *backoff.ExponentialBackOff {
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

	exp := getExponentialBackOff(3)

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

	os.Chmod(tempFile.Name(), 0666)

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
	exp := getExponentialBackOff(maxTimeout)

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

		output, err := docker.ExecCommandIntoContainer(context.Background(), host, "root", []string{"ps"})
		if err != nil {
			log.WithFields(log.Fields{
				"desiredState": desiredState,
				"elapsedTime":  exp.GetElapsedTime(),
				"error":        err,
				"host":         host,
				"process":      process,
				"retry":        retryCount,
			}).Warn("Could not execute process in the host")

			retryCount++

			return err
		}

		log.Debugf("Output: %s", output)

		outputContainsProcess := strings.Contains(output, process)

		// both true or both false
		if mustBePresent == outputContainsProcess {
			log.WithFields(log.Fields{
				"desiredState":  desiredState,
				"host":          host,
				"mustBePresent": outputContainsProcess,
				"process":       process,
				"ps":            output,
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
