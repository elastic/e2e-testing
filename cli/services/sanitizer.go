package services

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

// ConfigSanitizer sanitizes Beats services
type ConfigSanitizer interface {
	Sanitize(string) string
}

// GetConfigSanitizer returns the sanitizer for the service configuration, returning default
// if there is no sanitizer it will match sanitizers and services using service name
func GetConfigSanitizer(serviceType string) ConfigSanitizer {
	if strings.ToLower(serviceType) == "dropwizard" {
		return DropwizardSanitizer{}
	}

	return DefaultSanitizer{}
}

// DefaultSanitizer represents fallback sanitizer
type DefaultSanitizer struct{}

// Sanitize returns default content
func (s DefaultSanitizer) Sanitize(content string) string {
	log.Debug("Sanitising with default sanitiser: NOOP")
	return content
}

// DropwizardSanitizer represents a sanitizer for Dropwizard
type DropwizardSanitizer struct{}

// Sanitize prepends test application context
func (s DropwizardSanitizer) Sanitize(content string) string {
	log.Debug("Sanitising dropwizard")
	return strings.ReplaceAll(content, "metrics_path: /metrics/metrics", "metrics_path: /test/metrics")
}
