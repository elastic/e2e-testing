package main

import (
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// downloadFile will download a url and store it in a temporary path.
// It writes to the destination file as it downloads it, without
// loading the entire file into memory.
func downloadFile(url string) (string, error) {
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

	log.WithFields(log.Fields{
		"url":  url,
		"path": filepath,
	}).Debug("Downloading file")

	resp, err := http.Get(url)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"url":   url,
			"path":  filepath,
		}).Error("Error downloading file")
		return filepath, err
	}
	defer resp.Body.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"url":   url,
			"path":  filepath,
		}).Error("Error writing file")

		return filepath, err
	}

	log.WithFields(log.Fields{
		"url":  url,
		"path": filepath,
	}).Debug("File downloaded")

	return filepath, nil
}

func randomStringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randomString(length int) string {
	return randomStringWithCharset(length, charset)
}
