// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package services

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pact-foundation/pact-go/types"
)

// The Provider verification
func TestPactProvider(t *testing.T) {
	pact := createPact()

	// Verify the Provider - Tag-based Published Pacts for any known consumers
	_, err := pact.VerifyProvider(t, types.VerifyRequest{
		ProviderBaseURL:    fmt.Sprintf("http://127.0.0.1:%d", 5601),
		Tags:               []string{"7.9.1"},
		FailIfNoPactsFound: true,
		CustomProviderHeaders: []string{
			"Authorization: Basic ZWxhc3RpYzpjaGFuZ2VtZQ==",
			"Content-Type: application/json; charset=utf-8",
			"kbn-xsrf: provider-tests",
		},
		// Use this if you want to test without the Pact Broker
		PactURLs:                   []string{filepath.FromSlash(fmt.Sprintf("%s/e2e_testing_framework-fleet.json", os.Getenv("PACT_DIR")))},
		PublishVerificationResults: true,
		ProviderVersion:            "7.9.1",
	})

	if err != nil {
		t.Fatal(err)
	}
}
