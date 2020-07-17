// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"path"
	"testing"

	"github.com/Flaque/filet"
	"github.com/stretchr/testify/assert"
)

func TestMkdirAll(t *testing.T) {
	defer filet.CleanUp(t)

	tmpDir := filet.TmpDir(t, "")

	dir := path.Join(tmpDir, ".op", "compose", "services")

	err := MkdirAll(dir)
	assert.Nil(t, err)

	e, _ := Exists(dir)
	assert.True(t, e)
}
