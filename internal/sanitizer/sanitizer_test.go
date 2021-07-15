// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sanitizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetConfigSanitizer(t *testing.T) {
	tests := []struct {
		service         string
		content         string
		expectedContent string
	}{
		{
			service:         "compose",
			content:         `version: "2.3"`,
			expectedContent: `version: "3"`,
		},
		{
			service:         "dropwizard",
			content:         "metrics_path: /metrics/metrics",
			expectedContent: "metrics_path: /test/metrics",
		},
		{
			service:         "foo",
			content:         ": /metrics",
			expectedContent: ": /metrics",
		},
		{
			service:         "mssql",
			content:         `username: domain\username\n password: verysecurepassword`,
			expectedContent: `username: sa\n password: 1234_asdf`,
		},
		{
			service:         "mysql",
			content:         `hosts: ["root:secret@tcp(mysql:3306)/"]`,
			expectedContent: `hosts: ["root:test@tcp(mysql:3306)/"]`,
		},
	}

	for _, tt := range tests {
		ds := GetConfigSanitizer(tt.service)
		assert.Equal(t, tt.expectedContent, ds.Sanitize(tt.content))
	}
}
