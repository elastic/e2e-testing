package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testVersion = agentVersionBase

func TestCheckElasticAgentVersion(t *testing.T) {
	t.Run("Checking a version should return the version", func(t *testing.T) {
		v := checkElasticAgentVersion(testVersion, testVersion)

		assert.Equal(t, "8.0.0-SNAPSHOT", v)
	})

	t.Run("Should honour environment variable", func(t *testing.T) {
		defer os.Unsetenv("ELASTIC_AGENT_VERSION")
		os.Setenv("ELASTIC_AGENT_VERSION", "1.0.0")

		v := checkElasticAgentVersion(testVersion, testVersion)

		assert.Equal(t, "1.0.0", v)
	})

	t.Run("A PR should return base version", func(t *testing.T) {
		prVersion := "pr-12345"
		defer os.Unsetenv("ELASTIC_AGENT_VERSION")
		os.Setenv("ELASTIC_AGENT_VERSION", prVersion)

		v := checkElasticAgentVersion(prVersion, testVersion)

		assert.Equal(t, testVersion, v)
	})
}
