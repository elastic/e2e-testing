// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/e2e-testing/internal/elasticsearch"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
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
func (c *Client) GetAgentByHostname(ctx context.Context, hostname string) (Agent, error) {
	span, _ := apm.StartSpanOptions(ctx, "Getting Elastic Agent by hostname", "kibana.agent.get-by-hostname", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	agents, err := c.ListAgents(ctx)
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
func (c *Client) GetAgentIDByHostname(ctx context.Context, hostname string) (string, error) {
	agent, err := c.GetAgentByHostname(ctx, hostname)
	if err != nil {
		return "", err
	}
	log.WithFields(log.Fields{
		"agentId": agent.ID,
	}).Trace("Agent Id found")
	return agent.ID, nil
}

// GetAgentStatusByHostname gets agent status by hostname
func (c *Client) GetAgentStatusByHostname(ctx context.Context, hostname string) (string, error) {
	agentID, err := c.GetAgentIDByHostname(ctx, hostname)
	if err != nil {
		return "", err
	}

	span, _ := apm.StartSpanOptions(ctx, "Getting Elastic Agent status by hostname", "kibana.agent.get-status-by-hostname", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

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
func (c *Client) GetAgentEvents(ctx context.Context, applicationName string, agentID string, packagePolicyID string, updatedAt string) error {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []interface{}{
					map[string]interface{}{
						"bool": map[string]interface{}{
							"should": []interface{}{
								map[string]interface{}{
									"match": map[string]interface{}{
										"elastic_agent.id": agentID,
									},
								},
							},
							"minimum_should_match": 1,
						},
					},
					map[string]interface{}{
						"bool": map[string]interface{}{
							"should": []interface{}{
								map[string]interface{}{
									"match": map[string]interface{}{
										"data_stream.dataset": "elastic_agent",
									},
								},
							},
							"minimum_should_match": 1,
						},
					},
				},
			},
		},
	}

	indexName := "logs-elastic_agent-default"

	searchResult, err := elasticsearch.Search(ctx, indexName, query)
	if err != nil {
		log.WithFields(log.Fields{
			"agentID":         agentID,
			"application":     applicationName,
			"result":          searchResult,
			"error":           err,
			"packagePolicyID": packagePolicyID,
		}).Error("Could not get agent events from Fleet")
		return err
	}

	results := searchResult["hits"].(map[string]interface{})["hits"].([]interface{})

	for _, result := range results {
		if message, ok := result.(map[string]interface{})["_source"].(map[string]interface{})["message"].(string); ok {
			timestamp := result.(map[string]interface{})["_source"].(map[string]interface{})["@timestamp"].(string)
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
	}

	return fmt.Errorf("No %s events where found for the agent in the %s policy", applicationName, packagePolicyID)
}

// ListAgents returns the list of agents enrolled with Fleet.
func (c *Client) ListAgents(ctx context.Context) ([]Agent, error) {
	span, _ := apm.StartSpanOptions(ctx, "Listing Elastic Agents", "kibana.agents.list", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

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
func (c *Client) UnEnrollAgent(ctx context.Context, hostname string) error {
	agentID, err := c.GetAgentIDByHostname(ctx, hostname)
	if err != nil {
		return err
	}

	span, _ := apm.StartSpanOptions(ctx, "UnEnrolling Elastic Agent by hostname", "kibana.agent.un-enroll", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	reqBody := `{"revoke": true}`
	statusCode, respBody, _ := c.post(fmt.Sprintf("%s/agents/%s/unenroll", FleetAPI, agentID), []byte(reqBody))
	if statusCode != 200 {
		return fmt.Errorf("could not unenroll agent; API status code = %d, response body = %s", statusCode, respBody)
	}
	return nil
}

// UpgradeAgent upgrades an agent from to version
func (c *Client) UpgradeAgent(ctx context.Context, hostname string, version string) error {
	agentID, err := c.GetAgentIDByHostname(ctx, hostname)
	if err != nil {
		return err
	}

	span, _ := apm.StartSpanOptions(ctx, "Upgrading Elastic Agent by hostname", "kibana.agent.upgrade-by-hostname", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

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
