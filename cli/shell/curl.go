package shell

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// Get executes a GET request on a URL
func Get(url string) (*http.Response, error) {
	log.WithFields(log.Fields{
		"url": url,
	}).Debug("Executing GET request")

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return resp, nil
}

// Post executes a POST on a URL with a JSON payload as bytes
func Post(url string, payload []byte) error {
	log.WithFields(log.Fields{
		"url": url,
	}).Debug("Executing POST request")

	body := bytes.NewReader(payload)

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"url":   url,
		}).Error("Error creating POST request")
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"url":   url,
		}).Error("Error executing POST request")
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("Could not read response body")
		return err
	}
	bodyString := string(bodyBytes)

	// http.Status == 2xx
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		log.Debug(bodyString)
		return nil
	}

	return fmt.Errorf("POST request failed: %s", bodyString)
}
