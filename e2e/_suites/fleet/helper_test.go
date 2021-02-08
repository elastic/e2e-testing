package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testVersion = agentVersionBase

func TestCheckElasticAgentVersion(t *testing.T) {
	t.Run("Checking a version should return the version", func(t *testing.T) {
		v := checkElasticAgentVersion(testVersion, testVersion)

		assert.Equal(t, "8.0.0-SNAPSHOT", v)
	})

	t.Run("A PR should return base version", func(t *testing.T) {
		prVersion := "pr-12345"
		v := checkElasticAgentVersion(prVersion, testVersion)

		assert.Equal(t, testVersion, v)
	})
}
