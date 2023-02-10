// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package io

import (
	"path"
	"testing"

	"github.com/Flaque/filet"
	"github.com/stretchr/testify/assert"
)

func TestExists(t *testing.T) {
	t.Run("The file exists", func(t *testing.T) {
		f := path.Join("..", "_testresources", "dockerCopy.txt")

		b, err := Exists(f)
		assert.Nil(t, err)
		assert.True(t, b)
	})

	t.Run("The file does not exists", func(t *testing.T) {
		f := path.Join("..", "_testresources", "foo")

		b, err := Exists(f)
		assert.Nil(t, err)
		assert.False(t, b)
	})
}

func TestMkdirAll(t *testing.T) {
	defer filet.CleanUp(t)

	tmpDir := filet.TmpDir(t, "")

	dir := path.Join(tmpDir, ".op", "compose", "services")

	err := MkdirAll(dir)
	assert.Nil(t, err)

	e, _ := Exists(dir)
	assert.True(t, e)
}
