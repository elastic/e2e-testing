// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewElasticVersion(t *testing.T) {
	t.Run("newElasticVersion without git commit", func(t *testing.T) {
		v := newElasticVersion("1.2.3-SNAPSHOT")

		assert.Equal(t, "1.2.3", v.Version, "Version should not include SNAPSHOT nor commit")
		assert.Equal(t, "1.2.3-SNAPSHOT", v.FullVersion, "Full version should include SNAPSHOT")
		assert.Equal(t, "1.2.3", v.HashedVersion, "Hashed version should not include SNAPSHOT")
		assert.Equal(t, "1.2.3-SNAPSHOT", v.SnapshotVersion, "Snapshot version should include SNAPSHOT")
	})

	t.Run("newElasticVersion with git commit", func(t *testing.T) {
		v := newElasticVersion("1.2.3-abcdef-SNAPSHOT")

		assert.Equal(t, "1.2.3", v.Version, "Version should not include SNAPSHOT nor commit")
		assert.Equal(t, "1.2.3-abcdef-SNAPSHOT", v.FullVersion, "Full version should include commit and SNAPSHOT")
		assert.Equal(t, "1.2.3-abcdef", v.HashedVersion, "Hashed version should include commit but no SNAPSHOT")
		assert.Equal(t, "1.2.3-SNAPSHOT", v.SnapshotVersion, "Snapshot version should include SNAPSHOT but no commit")
	})
}

func Test_GetCommitVersion(t *testing.T) {
	t.Run("GetCommitVersion without git commit", func(t *testing.T) {
		v := GetCommitVersion("1.2.3-SNAPSHOT")

		assert.Equal(t, "1.2.3", v, "Version should not include SNAPSHOT nor commit")
	})

	t.Run("GetCommitVersion with git commit", func(t *testing.T) {
		v := GetCommitVersion("1.2.3-abcdef-SNAPSHOT")

		assert.Equal(t, "1.2.3-abcdef", v, "Version should not include SNAPSHOT nor commit")
	})
}

func Test_GetFullVersion(t *testing.T) {
	t.Run("GetFullVersion without git commit", func(t *testing.T) {
		v := GetFullVersion("1.2.3-SNAPSHOT")

		assert.Equal(t, "1.2.3-SNAPSHOT", v, "Version should not include SNAPSHOT nor commit")
	})

	t.Run("GetFullVersion with git commit", func(t *testing.T) {
		v := GetFullVersion("1.2.3-abcdef-SNAPSHOT")

		assert.Equal(t, "1.2.3-abcdef-SNAPSHOT", v, "Version should not include SNAPSHOT nor commit")
	})
}

func Test_GetSnapshotVersion(t *testing.T) {
	t.Run("GetSnapshotVersion without git commit", func(t *testing.T) {
		v := GetSnapshotVersion("1.2.3-SNAPSHOT")

		assert.Equal(t, "1.2.3-SNAPSHOT", v, "Version should include SNAPSHOT but no commit")
	})

	t.Run("GetCommitVersion with git commit", func(t *testing.T) {
		v := GetSnapshotVersion("1.2.3-abcdef-SNAPSHOT")

		assert.Equal(t, "1.2.3-SNAPSHOT", v, "Version should include SNAPSHOT but no commit")
	})
}

func Test_GetVersion(t *testing.T) {
	t.Run("GetVersion without git commit", func(t *testing.T) {
		v := GetVersion("1.2.3-SNAPSHOT")

		assert.Equal(t, "1.2.3", v, "Version should not include SNAPSHOT nor commit")
	})

	t.Run("GetVersion with git commit", func(t *testing.T) {
		v := GetVersion("1.2.3-abcdef-SNAPSHOT")

		assert.Equal(t, "1.2.3", v, "Version should not include SNAPSHOT nor commit")
	})
}
