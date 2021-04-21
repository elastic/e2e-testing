// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBaseURL(t *testing.T) {
	client, _ := NewClient()
	assert.NotNil(t, client)

	assert.Equal(t, "http://localhost:5601", client.host)
}

func TestNewClient(t *testing.T) {
	client, _ := NewClient()

	assert.NotNil(t, client)
}
