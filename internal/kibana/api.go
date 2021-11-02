// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

type ApiKey struct {
	APIKeys []APIKeys `json:"api_keys"`
}
type Metadata struct {
	PolicyID  string `json:"policy_id,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	ManagedBy string `json:"managed_by"`
	Managed   bool   `json:"managed"`
	Type      string `json:"type"`
}
type APIKeys struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Creation    int64    `json:"creation"`
	Invalidated bool     `json:"invalidated"`
	Username    string   `json:"username"`
	Realm       string   `json:"realm"`
	Metadata    Metadata `json:"metadata,omitempty"`
}

//  Get  _security/api_key
func (c *Client) ListApiKeys(ctx context.Context) ([]ApiKey, error) {
	span, _ := apm.StartSpanOptions(ctx, "Listing Api Keys", "fleet.agents.list", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	reqBody := `{}`
	statusCode, respBody, err := c.post(ctx, fmt.Sprintf("%s/_security/api_key?realm_name=_service_account&method=GET", ConsoleAPI), []byte(reqBody))

	if err != nil {
		log.WithFields(log.Fields{
			"body":  respBody,
			"error": err,
		}).Error("Could not get Api Keys")
		return nil, err
	}

	if statusCode != 200 {
		log.WithFields(log.Fields{
			"body":       respBody,
			"error":      err,
			"statusCode": statusCode,
		}).Error("Could not get Api Keys")

		return nil, err
	}
	var resp struct {
		List []ApiKey `json:"list"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, "could not convert list Api Keys (response) to JSON")
	}

	return resp.List, nil
}
