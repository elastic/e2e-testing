// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package shell

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

		val, err := GetEnvBool(test.key)

		if test.key == "should_be_empty" {
			assert.NotNil(t, err)
			assert.False(t, val)
		} else if test.key == "should_be_not_empty" {
			assert.Nil(t, err)
			assert.False(t, val)
		} else if test.key == "should_be_true" {
			assert.Nil(t, err)
			assert.True(t, val)
		} else if test.key == "should_be_false" {
			assert.Nil(t, err)
			assert.False(t, val)
		}
	}
}
