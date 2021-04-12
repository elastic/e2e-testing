// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Agent represents an Elastic Agent enrolled with fleet.
type Agent struct {
	ID             string `json:"id"`
	PolicyID       string `json:"policy_id"`
	PolicyRevision int    `json:"policy_revision,omitempty"`
	LocalMetadata  struct {
		Host struct {
			Name     string `json:"name"`
			HostName string `json:"hostname"`
		} `json:"host"`
		Elastic struct {
			Agent struct {
				Version  string `json:"version"`
				Snapshot bool   `json:"snapshot"`
			} `json:"agent"`
		} `json:"elastic"`
	} `json:"local_metadata"`
	Status string `json:"status"`
}

// GetAgentByHostname get an agent by the local_metadata.host.name property
func (c *Client) GetAgentByHostname(hostname string) (Agent, error) {
	agents, err := c.ListAgents()
	if err != nil {
		return Agent{}, err
	}

	for _, agent := range agents {
		agentHostname := agent.LocalMetadata.Host.Name
		// a hostname has an agentID by status
		if agentHostname == hostname {
			log.WithFields(log.Fields{
				"agent": agent,
			}).Trace("Agent found")
			return agent, nil
		}
	}

	return Agent{}, nil
}

// GetAgentIDByHostname gets agent id by hostname
func (c *Client) GetAgentIDByHostname(hostname string) (string, error) {
	agent, err := c.GetAgentByHostname(hostname)
	if err != nil {
		return "", err
	}
	log.WithFields(log.Fields{
		"agentId": agent.ID,
	}).Trace("Agent Id found")
	return agent.ID, nil
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

	var resp struct {
		Item Agent `json:"item"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", errors.Wrap(err, "could not convert list agents (response) to JSON")
	}

	log.WithFields(log.Fields{
		"agentStatus": resp.Item.Status,
	}).Trace("Agent Status found")
	return resp.Item.Status, nil
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

// ListAgents returns the list of agents enrolled with Fleet.
func (c *Client) ListAgents() ([]Agent, error) {
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

	var resp struct {
		List []Agent `json:"list"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, "could not convert list agents (response) to JSON")
	}

	return resp.List, nil

}

// UnEnrollAgent unenrolls agent from fleet
func (c *Client) UnEnrollAgent(hostname string, force bool) error {
	agentID, err := c.GetAgentIDByHostname(hostname)
	if err != nil {
		return err
	}
	reqBody := `{"force": false}`
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
