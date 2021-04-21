// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Client is responsible for exporting dashboards from Kibana.
type Client struct {
	host     string
	username string
	password string
}

// NewClient creates a new instance of the client.
func NewClient() (*Client, error) {
	host := BaseURL
	username := "elastic"
	password := "changeme"

	return &Client{
		host:     host,
		username: username,
		password: password,
	}, nil
}

func (c *Client) get(resourcePath string) (int, []byte, error) {
	return c.sendRequest(http.MethodGet, resourcePath, nil)
}

func (c *Client) post(resourcePath string, body []byte) (int, []byte, error) {
	return c.sendRequest(http.MethodPost, resourcePath, body)
}

func (c *Client) put(resourcePath string, body []byte) (int, []byte, error) {
	return c.sendRequest(http.MethodPut, resourcePath, body)
}

func (c *Client) delete(resourcePath string) (int, []byte, error) {
	return c.sendRequest(http.MethodDelete, resourcePath, nil)
}

func (c *Client) sendRequest(method, resourcePath string, body []byte) (int, []byte, error) {
	reqBody := bytes.NewReader(body)
	base, err := url.Parse(c.host)
	if err != nil {
		return 0, nil, errors.Wrapf(err, "could not create base URL from host: %v", c.host)
	}

	rel, err := url.Parse(resourcePath)
	if err != nil {
		return 0, nil, errors.Wrapf(err, "could not create relative URL from resource path: %v", resourcePath)
	}

	u := base.ResolveReference(rel)

	log.WithFields(log.Fields{
		"method": method,
		"url":    u,
	}).Trace("Kibana API Query")

	req, err := http.NewRequest(method, u.String(), reqBody)
	if err != nil {
		return 0, nil, errors.Wrapf(err, "could not create %v request to Kibana API resource: %s", method, resourcePath)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("kbn-xsrf", "e2e-tests")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not send request to Kibana API")
	}

	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, errors.Wrap(err, "could not read response body")
	}

	return resp.StatusCode, body, nil
}
