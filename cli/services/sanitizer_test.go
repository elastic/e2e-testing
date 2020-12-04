package services

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
			service:         "dropwizard",
			content:         "metrics_path: /metrics/metrics",
			expectedContent: "metrics_path: /test/metrics",
		},
		{
			service:         "foo",
			content:         ": /metrics",
			expectedContent: ": /metrics",
		},
	}

	for _, tt := range tests {
		ds := GetConfigSanitizer(tt.service)
		assert.Equal(t, tt.expectedContent, ds.Sanitize(tt.content))
	}
}
