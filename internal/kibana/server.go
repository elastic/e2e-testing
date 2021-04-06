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
	"github.com/google/uuid"
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
func (c *Client) CreateEnrollmentAPIKey(policy Policy) (EnrollmentAPIKey, error) {
	uuid := uuid.New().String()

	reqBody := `{"policy_id": "` + policy.ID + `", "name": "Test token for ` + policy.Name + `-` + uuid + `"}`
	statusCode, respBody, _ := c.post(fmt.Sprintf("%s/enrollment-api-keys", FleetAPI), []byte(reqBody))
	if statusCode != 200 {
		jsonParsed, err := gabs.ParseJSON([]byte(respBody))
		log.WithFields(log.Fields{
			"body":       jsonParsed,
			"reqBody":    reqBody,
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
func (c *Client) DeleteEnrollmentAPIKey(enrollmentID string) error {
	statusCode, respBody, err := c.delete(fmt.Sprintf("%s/enrollment-api-keys/%s", FleetAPI, enrollmentID))

	if err != nil {
		log.WithFields(log.Fields{
			"body":  respBody,
			"error": err,
		}).Error("Could not delete enrollment key")
		return err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"body":       respBody,
			"error":      err,
			"statusCode": statusCode,
		}).Error("Could not delete enrollment key")

		return err
	}
	return nil
}

// ListEnrollmentAPIKeys list the enrollment api keys
func (c *Client) ListEnrollmentAPIKeys() ([]EnrollmentAPIKey, error) {
	statusCode, respBody, err := c.get(fmt.Sprintf("%s/enrollment-api-keys", FleetAPI))

	if err != nil {
		log.WithFields(log.Fields{
			"body":  respBody,
			"error": err,
		}).Error("Could not get Integration package")
		return []EnrollmentAPIKey{}, err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"body":       respBody,
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
func (c *Client) RecreateFleet() error {
	waitForFleet := func() error {
		reqBody := `{ "forceRecreate": true }`
		statusCode, respBody, err := c.post(fmt.Sprintf("%s/agents/setup", FleetAPI), []byte(reqBody))
		if err != nil {
			log.WithFields(log.Fields{
				"body":       respBody,
				"error":      err,
				"statusCode": statusCode,
			}).Error("Could not initialise Fleet setup")
			return err
		}

		jsonResponse, err := gabs.ParseJSON([]byte(respBody))
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
	err := backoff.Retry(waitForFleet, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 10))
	if err != nil {
		return err
	}
	return nil
}

// WaitForFleet waits for fleet server to be ready
func (c *Client) WaitForFleet() error {
	waitForFleet := func() error {
		statusCode, respBody, err := c.get(fmt.Sprintf("%s/agents/setup", FleetAPI))
		if err != nil {
			log.WithFields(log.Fields{
				"body":       respBody,
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

		jsonResponse, err := gabs.ParseJSON([]byte(respBody))
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
			}).Error("Kibana has not been initialized")
			return errors.New("Kibana has not been initialized")
		}
		log.Info("Kibana setup initialized")
		return nil
	}
	err := backoff.Retry(waitForFleet, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 15))
	if err != nil {
		return err
	}
	return nil

}

// WaitForReady waits for Kibana to be healthy and accept connections
func (c *Client) WaitForReady(maxTimeoutMinutes time.Duration) (bool, error) {
	var (
		initialInterval     = 500 * time.Millisecond
		randomizationFactor = 0.5
		multiplier          = 2.0
		maxInterval         = 5 * time.Second
		maxElapsedTime      = maxTimeoutMinutes
	)

	ctx := context.Background()
	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = initialInterval
	exp.RandomizationFactor = randomizationFactor
	exp.Multiplier = multiplier
	exp.MaxInterval = maxInterval
	exp.MaxElapsedTime = maxElapsedTime

	retryCount := 1

	kibanaStatus := func() error {
		span, _ := apm.StartSpanOptions(ctx, "Health", "kibana.health", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		defer span.End()

		statusCode, respBody, err := c.get("status")
		if err != nil {
			log.WithFields(log.Fields{
				"error":          err,
				"statusCode":     statusCode,
				"respBody":       respBody,
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

// GetDataStreams get data streams from deployed agents
func (c *Client) GetDataStreams() (*gabs.Container, error) {
	statusCode, respBody, err := c.get(fmt.Sprintf("%s/data_streams", FleetAPI))

	if err != nil {
		log.WithFields(log.Fields{
			"body":  respBody,
			"error": err,
		}).Error("Could not get Fleet data streams")
		return &gabs.Container{}, err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"body":       respBody,
			"error":      err,
			"statusCode": statusCode,
		}).Error("Could not get Fleet data streams api")

		return &gabs.Container{}, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(respBody))
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
