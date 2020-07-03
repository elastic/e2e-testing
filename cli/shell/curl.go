package shell

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	log "github.com/sirupsen/logrus"
)

// HTTPRequest configures an HTTP request
type HTTPRequest struct {
	BasicAuthUser     string
	BasicAuthPassword string
	EncodeURL         bool
	Headers           map[string]string
	method            string
	Payload           []byte
	QueryString       string
	URL               string
}

// GetURL returns the URL as a string
func (req *HTTPRequest) GetURL() string {
	if req.QueryString == "" {
		return req.URL
	}

	u := req.URL + "?"
	if req.EncodeURL {
		return u + url.QueryEscape(req.QueryString)
	}

	return u + req.QueryString
}

// Get executes a GET request
func Get(r HTTPRequest) (string, error) {
	r.method = "GET"

	return request(r)
}

// Post executes a POST request
func Post(r HTTPRequest) (string, error) {
	r.method = "POST"

	return request(r)
}

// Post executes a request
func request(r HTTPRequest) (string, error) {
	escapedURL := r.GetURL()

	log.WithFields(log.Fields{
		"method":     r.method,
		"escapedURL": escapedURL,
	}).Debug("Executing request")

	var body io.Reader
	if r.Payload != nil {
		body = bytes.NewReader(r.Payload)
	} else {
		body = nil
	}

	req, err := http.NewRequest(r.method, escapedURL, body)
	if err != nil {
		log.WithFields(log.Fields{
			"error":      err,
			"method":     r.method,
			"escapedURL": escapedURL,
		}).Warn("Error creating request")
		return "", err
	}

	if r.Headers != nil {
		for k, v := range r.Headers {
			req.Header.Set(k, v)
		}
	}

	if r.BasicAuthUser != "" {
		req.SetBasicAuth(r.BasicAuthUser, r.BasicAuthPassword)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"error":      err,
			"method":     r.method,
			"escapedURL": escapedURL,
		}).Warn("Error executing request")
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"error":      err,
			"method":     r.method,
			"escapedURL": escapedURL,
		}).Warn("Could not read response body")
		return "", err
	}
	bodyString := string(bodyBytes)

	// http.Status ==> [2xx, 4xx)
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest {
		return bodyString, nil
	}

	return bodyString, fmt.Errorf("%s request failed with %d", r.method, resp.StatusCode)
}
