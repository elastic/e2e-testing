// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
)

// UnEnrollAgent unenrolls agent from fleet
func (c *Client) UnEnrollAgent(hostname string, force bool) error {
	agentID, err := c.GetAgentIDByHostname(hostname)
	if err != nil {
		return err
	}
	reqBody := `{}`
	if force {
		reqBody = `{"force": true}`
	}
	statusCode, respBody, _ := c.post(fmt.Sprintf("%s/agents/%s/unenroll", FleetAPI, agentID), []byte(reqBody))
	if statusCode != 200 {
		return fmt.Errorf("could not unenroll agent; API status code = %d, response body = %s", statusCode, respBody)
	}
	return nil
}

// UpgradeAgent upgrades an agent from to version
func (c *Client) UpgradeAgent(hostname string, version string) error {
	agentID, err := c.GetAgentIDByHostname(hostname)
	if err != nil {
		return err
	}
	reqBody := `{"version":"` + version + `", "force": true}`
	statusCode, respBody, err := c.post(fmt.Sprintf("%s/agents/%s/upgrade", FleetAPI, agentID), []byte(reqBody))
	if statusCode != 200 {
		log.WithFields(log.Fields{
			"body":       respBody,
			"error":      err,
			"statusCode": statusCode,
		}).Error("Could not upgrade agent")

		return err
	}
	return nil

}

// ListAgents returns the list of agents enrolled with Fleet.
func (c *Client) ListAgents() (*gabs.Container, error) {
	statusCode, respBody, err := c.get(fmt.Sprintf("%s/agents", FleetAPI))

	if err != nil {
		log.WithFields(log.Fields{
			"body":  respBody,
			"error": err,
		}).Error("Could not get Fleet's online agents")
		return nil, err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"body":       respBody,
			"error":      err,
			"statusCode": statusCode,
		}).Error("Could not get Fleet's online agents")

		return nil, err
	}

	jsonResponse, err := gabs.ParseJSON([]byte(respBody))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"body":  respBody,
		}).Error("Could not parse response into JSON")
		return nil, err
	}

	return jsonResponse, nil
}

// GetAgentByHostname get an agent by the local_metadata.host.name property
func (c *Client) GetAgentByHostname(hostname string) (*gabs.Container, error) {
	jsonParsed, err := c.ListAgents()
	if err != nil {
		return jsonParsed, err
	}

	hosts := jsonParsed.Path("list").Children()

	for _, host := range hosts {
		agentHostname := host.Path("local_metadata.host.hostname").Data().(string)
		// a hostname has an agentID by status
		if agentHostname == hostname {
			log.WithFields(log.Fields{
				"agent": host,
			}).Trace("Agent found")
			return host, nil
		}
	}

	return jsonParsed, nil
}

// GetAgentIDByHostname gets agent id by hostname
func (c *Client) GetAgentIDByHostname(hostname string) (string, error) {
	jsonParsed, err := c.GetAgentByHostname(hostname)
	if err != nil {
		return "", err
	}
	agentID := jsonParsed.Path("id").Data().(string)
	log.WithFields(log.Fields{
		"agentId": agentID,
	}).Trace("Agent Id found")
	return agentID, nil
}

// GetAgentStatusByHostname gets agent status by hostname
func (c *Client) GetAgentStatusByHostname(hostname string) (string, error) {
	agentID, err := c.GetAgentIDByHostname(hostname)
	if err != nil {
		return "", err
	}

	statusCode, respBody, err := c.get(fmt.Sprintf("%s/agents/%s", FleetAPI, agentID))
	if err != nil {
		log.WithFields(log.Fields{
			"body":       respBody,
			"error":      err,
			"statusCode": statusCode,
		}).Error("Could not get agent response")
		return "", err
	}

	jsonResponse, err := gabs.ParseJSON([]byte(respBody))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"body":  respBody,
		}).Error("Could not parse response into JSON")
		return "", err
	}

	agentStatus := jsonResponse.Path("item.status").Data().(string)
	log.WithFields(log.Fields{
		"agentStatus": agentStatus,
	}).Trace("Agent Status found")
	return agentStatus, nil
}

// GetAgentEvents get events of agent
func (c *Client) GetAgentEvents(applicationName string, agentID string, packagePolicyID string, updatedAt string) error {
	statusCode, respBody, err := c.get(fmt.Sprintf("%s/agents/%s/events", FleetAPI, agentID))

	if err != nil {
		log.WithFields(log.Fields{
			"agentID":         agentID,
			"application":     applicationName,
			"body":            respBody,
			"error":           err,
			"packagePolicyID": packagePolicyID,
		}).Error("Could not get agent events from Fleet")

		return err
	}

	jsonResponse, err := gabs.ParseJSON([]byte(respBody))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": jsonResponse,
		}).Error("Could not parse response into JSON")
		return err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"agentID":         agentID,
			"application":     applicationName,
			"body":            jsonResponse,
			"error":           err,
			"packagePolicyID": packagePolicyID,
		}).Error("Could not get agent events from Fleet")

		return err
	}

	listItems := jsonResponse.Path("list").Children()
	for _, item := range listItems {
		message := item.Path("message").Data().(string)
		// we use a string because we are not able to process what comes in the event, so we will do
		// an alphabetical order, as they share same layout but different millis and timezone format
		timestamp := item.Path("timestamp").Data().(string)

		log.WithFields(log.Fields{
			"agentID":         agentID,
			"application":     applicationName,
			"event_at":        timestamp,
			"message":         message,
			"packagePolicyID": packagePolicyID,
			"updated_at":      updatedAt,
		}).Trace("Event found")

		matches := (strings.Contains(message, applicationName) &&
			strings.Contains(message, "["+agentID+"]: State changed to") &&
			strings.Contains(message, "Protecting with policy {"+packagePolicyID+"}"))

		if matches && timestamp > updatedAt {
			log.WithFields(log.Fields{
				"application":     applicationName,
				"event_at":        timestamp,
				"packagePolicyID": packagePolicyID,
				"updated_at":      updatedAt,
				"message":         message,
			}).Info("Event after the update was found")
			return nil
		}
	}

	return fmt.Errorf("No %s events where found for the agent in the %s policy", applicationName, packagePolicyID)
}
