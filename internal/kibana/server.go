// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// EnrollmentAPIKey struct for holding enrollment response
type EnrollmentAPIKey struct {
	Active   bool   `json:"active"`
	APIKey   string `json:"api_key"`
	APIKeyID string `json:"api_key_id"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	PolicyID string `json:"policy_id"`
}

// CreateEnrollmentAPIKey creates an enrollment api key
func (c *Client) CreateEnrollmentAPIKey(ctx context.Context, policy Policy) (EnrollmentAPIKey, error) {
	span, _ := apm.StartSpanOptions(ctx, "Creating enrollment API Key", "fleet.api-key.create", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	reqBody := `{"policy_id": "` + policy.ID + `"}`
	statusCode, respBody, _ := c.post(ctx, fmt.Sprintf("%s/enrollment-api-keys", FleetAPI), []byte(reqBody))
	if statusCode != 200 {
		jsonParsed, err := gabs.ParseJSON(respBody)
		log.WithFields(log.Fields{
			"body":       jsonParsed,
			"reqBody":    string(reqBody),
			"error":      err,
			"statusCode": statusCode,
		}).Error("Could not create enrollment api key")

		return EnrollmentAPIKey{}, err
	}

	var resp struct {
		Enrollment EnrollmentAPIKey `json:"item"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return EnrollmentAPIKey{}, errors.Wrap(err, "Unable to convert enrollment response to JSON")
	}

	return resp.Enrollment, nil
}

// DeleteEnrollmentAPIKey deletes the enrollment api key
func (c *Client) DeleteEnrollmentAPIKey(ctx context.Context, enrollmentID string) error {
	span, _ := apm.StartSpanOptions(ctx, "Deleting enrollment API Key", "fleet.api-key.delete", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	statusCode, respBody, err := c.delete(ctx, fmt.Sprintf("%s/enrollment-api-keys/%s", FleetAPI, enrollmentID))

	if err != nil {
		log.WithFields(log.Fields{
			"body":  string(respBody),
			"error": err,
		}).Error("Could not delete enrollment key")
		return err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"body":       string(respBody),
			"error":      err,
			"statusCode": statusCode,
		}).Error("Could not delete enrollment key")

		return err
	}
	return nil
}

// GetDataStreams get data streams from deployed agents
func (c *Client) GetDataStreams(ctx context.Context) (*gabs.Container, error) {
	span, _ := apm.StartSpanOptions(ctx, "Getting data streams", "kibana.data-streams.list", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	statusCode, respBody, err := c.get(ctx, fmt.Sprintf("%s/data_streams", FleetAPI))

	if err != nil {
		log.WithFields(log.Fields{
			"body":  string(respBody),
			"error": err,
		}).Error("Could not get Fleet data streams")
		return &gabs.Container{}, err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"body":       string(respBody),
			"error":      err,
			"statusCode": statusCode,
		}).Error("Could not get Fleet data streams api")

		return &gabs.Container{}, err
	}

	jsonParsed, err := gabs.ParseJSON(respBody)
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": jsonParsed,
		}).Error("Could not parse response into JSON")
		return nil, err
	}

	// data streams should contain array of elements
	dataStreams := jsonParsed.Path("data_streams")

	log.WithFields(log.Fields{
		"count": len(dataStreams.Children()),
	}).Debug("Data Streams retrieved")

	return dataStreams, nil
}

// ListEnrollmentAPIKeys list the enrollment api keys
func (c *Client) ListEnrollmentAPIKeys(ctx context.Context) ([]EnrollmentAPIKey, error) {
	span, _ := apm.StartSpanOptions(ctx, "Listing enrollment API Keys", "fleet.api-keys.list", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	statusCode, respBody, err := c.get(ctx, fmt.Sprintf("%s/enrollment-api-keys", FleetAPI))

	if err != nil {
		log.WithFields(log.Fields{
			"body":  string(respBody),
			"error": err,
		}).Error("Could not get Integration package")
		return []EnrollmentAPIKey{}, err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"body":       string(respBody),
			"error":      err,
			"statusCode": statusCode,
		}).Error("Could not get enrollment apis")

		return []EnrollmentAPIKey{}, err
	}

	var resp struct {
		List []EnrollmentAPIKey `json:"list"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, "Unable to convert list of enrollment apis to JSON")
	}

	return resp.List, nil

}

// RecreateFleet this will force recreate the fleet configuration
func (c *Client) RecreateFleet(ctx context.Context) error {
	waitForFleet := func() error {
		span, _ := apm.StartSpanOptions(ctx, "Recreating Fleet configuration", "fleet.config.recreate", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		defer span.End()

		reqBody := `{ "forceRecreate": true }`
		statusCode, respBody, err := c.post(ctx, fmt.Sprintf("%s/setup", FleetAPI), []byte(reqBody))
		if err != nil {
			log.WithFields(log.Fields{
				"body":       string(respBody),
				"error":      err,
				"statusCode": statusCode,
			}).Error("Could not initialise Fleet setup")
			return err
		}

		jsonResponse, err := gabs.ParseJSON(respBody)
		if err != nil {
			log.WithFields(log.Fields{
				"body":       jsonResponse,
				"error":      err,
				"statusCode": statusCode,
			}).Error("Could not parse JSON response")
			return err
		}

		if statusCode != 200 {
			log.WithFields(log.Fields{
				"statusCode": statusCode,
				"body":       jsonResponse,
			}).Warn("Fleet not ready")
			return errors.New("Fleet not ready")
		}

		log.WithFields(log.Fields{
			"body":       jsonResponse,
			"statusCode": statusCode,
		}).Info("Fleet setup done")
		return nil
	}
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	exp := utils.GetExponentialBackOff(maxTimeout)

	err := backoff.Retry(waitForFleet, exp)
	if err != nil {
		return err
	}
	return nil
}

// WaitForFleet waits for fleet server to be ready
func (c *Client) WaitForFleet(ctx context.Context) error {
	waitForFleet := func() error {
		span, _ := apm.StartSpanOptions(ctx, "Fleet setup", "kibana.fleet.setup", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		defer span.End()

		statusCode, respBody, err := c.get(ctx, fmt.Sprintf("%s/agents/setup", FleetAPI))
		if err != nil {
			log.WithFields(log.Fields{
				"body":       string(respBody),
				"error":      err,
				"statusCode": statusCode,
			}).Error("Could not verify Fleet is setup and ready")
			return err
		}
		if statusCode != 200 {
			log.WithFields(log.Fields{
				"statusCode": statusCode,
			}).Warn("Fleet not ready")
			return err
		}

		jsonResponse, err := gabs.ParseJSON(respBody)
		if err != nil {
			log.WithFields(log.Fields{
				"body":       jsonResponse,
				"error":      err,
				"statusCode": statusCode,
			}).Error("Could not parse JSON response")
			return err
		}

		isReady := jsonResponse.Path("isReady").Data().(bool)
		if !isReady {
			log.WithFields(log.Fields{
				"body":       jsonResponse,
				"error":      err,
				"statusCode": statusCode,
			}).Warn("Fleet is not ready")
			return errors.New("Fleet is not ready")
		}
		log.Info("Fleet setup complete")
		return nil
	}
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	exp := utils.GetExponentialBackOff(maxTimeout)

	err := backoff.Retry(waitForFleet, exp)
	if err != nil {
		return err
	}
	return nil

}

// WaitForReady waits for Kibana to be healthy and accept connections
func (c *Client) WaitForReady(ctx context.Context, maxTimeoutMinutes time.Duration) (bool, error) {
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	exp := utils.GetExponentialBackOff(maxTimeout)

	retryCount := 1

	kibanaStatus := func() error {
		span, _ := apm.StartSpanOptions(ctx, "Health", "kibana.health", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		defer span.End()

		statusCode, respBody, err := c.get(ctx, "status")
		if err != nil {
			log.WithFields(log.Fields{
				"error":          err,
				"statusCode":     statusCode,
				"respBody":       string(respBody),
				"retry":          retryCount,
				"statusEndpoint": fmt.Sprintf("%s/status", BaseURL),
				"elapsedTime":    exp.GetElapsedTime(),
			}).Warn("The Kibana instance is not healthy yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"retries":        retryCount,
			"statusEndpoint": fmt.Sprintf("%s/status", BaseURL),
			"elapsedTime":    exp.GetElapsedTime(),
		}).Info("The Kibana instance is healthy")

		return nil
	}

	err := backoff.Retry(kibanaStatus, exp)
	if err != nil {
		return false, err
	}

	return true, nil
}
