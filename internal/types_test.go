// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetArchitecture(t *testing.T) {
	t.Run("Retrieving amd architecture", func(t *testing.T) {
		fallbackArch := os.Getenv("GOARCH")
		os.Setenv("GOARCH", "x86_64")
		defer os.Setenv("GOARCH", fallbackArch)

		arch := GetArchitecture()
		assert.Equal(t, Amd64, arch)
	})

	t.Run("Retrieving amd architecture as fallback", func(t *testing.T) {
		fallbackArch := os.Getenv("GOARCH")
		os.Setenv("GOARCH", "arch-not-found")
		defer os.Setenv("GOARCH", fallbackArch)

		arch := GetArchitecture()
		assert.Equal(t, Amd64, arch)
	})

	t.Run("Retrieving arm architecture", func(t *testing.T) {
		fallbackArch := os.Getenv("GOARCH")
		os.Setenv("GOARCH", "arm64")
		defer os.Setenv("GOARCH", fallbackArch)

		arch := GetArchitecture()
		assert.Equal(t, Arm64, arch)
	})
}
