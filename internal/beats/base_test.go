// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beats

import (
	"testing"

	"github.com/elastic/e2e-testing/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestNewBeat(t *testing.T) {
	t.Run("Create no x-pack Beat", func(t *testing.T) {
		b := GenericBeat("beatless", types.Linux, types.Amd64, types.Deb, "THE-VERSION").NoXPack()

		assert.False(t, b.XPack)
	})

	t.Run("Create ARM64 binaries from RPM", func(t *testing.T) {
		b := GenericBeat("beatless", types.Linux, types.Arm64, types.Rpm, "THE-VERSION")

		assert.Equal(t, types.Aarch64, b.Arch)
	})
	t.Run("Create ARM64 binaries from non-RPM", func(t *testing.T) {
		b := GenericBeat("beatless", types.Linux, types.Arm64, types.Deb, "THE-VERSION")

		assert.Equal(t, types.Arm64, b.Arch)
	})

	t.Run("Cannot create docker images from DEB", func(t *testing.T) {
		b := GenericBeat("beatless", types.Linux, types.Amd64, types.Deb, "THE-VERSION").AsDocker()

		assert.False(t, b.Docker)
	})

	t.Run("Cannot create docker images from RPM", func(t *testing.T) {
		b := GenericBeat("beatless", types.Linux, types.Amd64, types.Rpm, "THE-VERSION").AsDocker()

		assert.False(t, b.Docker)
	})

	t.Run("Can create docker images from TAR on AMD", func(t *testing.T) {
		b := GenericBeat("beatless", types.Linux, types.Amd64, types.TarGz, "THE-VERSION").AsDocker()

		assert.True(t, b.Docker)
	})
	t.Run("Can create docker images from TAR on ARM", func(t *testing.T) {
		b := GenericBeat("beatless", types.Linux, types.Arm64, types.TarGz, "THE-VERSION").AsDocker()

		assert.True(t, b.Docker)
	})

	t.Run("Can create docker images from ZIP", func(t *testing.T) {
		b := NewWindowsBeat("beatless", "THE-VERSION", true).AsDocker()

		assert.True(t, b.Docker)
	})
	t.Run("Cannot create docker images from Windows and not ZIP", func(t *testing.T) {
		b := GenericBeat("beatless", types.Windows, types.Amd64, types.Rpm, "THE-VERSION").AsDocker()

		assert.False(t, b.Docker)
	})
}
