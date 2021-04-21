// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package shell

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	t.Run("Empty value should return fallback", func(t *testing.T) {
		defer os.Unsetenv("test.key")
		os.Setenv("test.key", "")

		val := GetEnv("test.key", "fallback")
		assert.Equal(t, "fallback", val)
	})

	t.Run("Non existing key should return fallback", func(t *testing.T) {
		val := GetEnv("test.key", "fallback")
		assert.Equal(t, "fallback", val)
	})

	t.Run("Value should return value", func(t *testing.T) {
		defer os.Unsetenv("test.key")
		os.Setenv("test.key", "value")

		val := GetEnv("test.key", "fallback")
		assert.Equal(t, "value", val)
	})
}

func TestGetEnvBool(t *testing.T) {
	type test struct {
		key   string
		value string
	}

	tests := []test{
		{key: "should_be_true", value: "true"},
		{key: "should_be_false", value: "false"},
		{key: "should_be_empty", value: ""},
		{key: "should_be_not_empty", value: "garbage"},
	}

	for _, test := range tests {
		os.Setenv(test.key, test.value)

		val := GetEnvBool(test.key)

		switch test.key {
		case "should_be_empty":
			assert.False(t, val)
		case "should_be_not_empty":
			assert.False(t, val)
		case "should_be_true":
			assert.True(t, val)
		case "should_be_false":
			assert.False(t, val)
		}
	}
}
