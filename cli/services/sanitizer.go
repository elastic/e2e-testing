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
	if strings.ToLower(serviceType) == "compose" {
		return DockerComposeSanitizer{}
	} else if strings.ToLower(serviceType) == "dropwizard" {
		return DropwizardSanitizer{}
	} else if strings.ToLower(serviceType) == "mysql" {
		return MySQLSanitizer{}
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

// DockerComposeSanitizer represents a sanitizer for Docker Compose
type DockerComposeSanitizer struct{}

// Sanitize upgrades compose versions to v3, because v2 does not support extends
func (s DockerComposeSanitizer) Sanitize(content string) string {
	log.Debug("Sanitising Docker Compose")
	return strings.ReplaceAll(content, `version: "2.3"`, `version: "3"`)
}

// DropwizardSanitizer represents a sanitizer for Dropwizard
type DropwizardSanitizer struct{}

// Sanitize prepends test application context
func (s DropwizardSanitizer) Sanitize(content string) string {
	log.Debug("Sanitising dropwizard")
	return strings.ReplaceAll(content, "metrics_path: /metrics/metrics", "metrics_path: /test/metrics")
}

// MySQLSanitizer represents a sanitizer for Dropwizard
type MySQLSanitizer struct{}

// Sanitize prepends test application context
func (s MySQLSanitizer) Sanitize(content string) string {
	log.Debug("Sanitising mysql")
	return strings.ReplaceAll(content, "root:secret", "root:test")
}
