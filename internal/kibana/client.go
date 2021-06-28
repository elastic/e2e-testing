// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// Client is responsible for exporting dashboards from Kibana.
type Client struct {
	host     string
	username string
	password string
}

// NewClient creates a new instance of the client.
func NewClient() (*Client, error) {
	host := getBaseURL()
	username := "elastic"
	password := "changeme"

	return &Client{
		host:     host,
		username: username,
		password: password,
	}, nil
}

func (c *Client) get(ctx context.Context, resourcePath string) (int, []byte, error) {
	return c.sendRequest(ctx, http.MethodGet, resourcePath, nil)
}

func (c *Client) post(ctx context.Context, resourcePath string, body []byte) (int, []byte, error) {
	return c.sendRequest(ctx, http.MethodPost, resourcePath, body)
}

func (c *Client) put(ctx context.Context, resourcePath string, body []byte) (int, []byte, error) {
	return c.sendRequest(ctx, http.MethodPut, resourcePath, body)
}

func (c *Client) delete(ctx context.Context, resourcePath string) (int, []byte, error) {
	return c.sendRequest(ctx, http.MethodDelete, resourcePath, nil)
}

func (c *Client) sendRequest(ctx context.Context, method, resourcePath string, body []byte) (int, []byte, error) {
	span, _ := apm.StartSpanOptions(ctx, "Sending HTTP request", "http.request."+method, apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("method", method)
	span.Context.SetLabel("base", c.host)
	span.Context.SetLabel("resourcePath", resourcePath)
	defer span.End()

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
