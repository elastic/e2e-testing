// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ExtractZip(t *testing.T) {
	src := filepath.Join("..", "_testresources", "sample.zip")
	//src := filepath.Join("/tmp", "elastic-agent-8.4.0-SNAPSHOT-windows-x86_64.zip")
	tmp := t.TempDir()
	target := filepath.Join(tmp, "sample")

	err := extractZIPFile(src, target)
	assert.Nil(t, err)

	files := 0
	dirs := 0
	expectedFiles := 2 // sample.tar.gz, dockerCopy.txt
	expectedDirs := 3  // sample, internal, _testresources

	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			dirs++
		} else {
			files++
		}
		return nil
	})
	assert.Nil(t, err)

	assert.Equal(t, expectedFiles, files)
	assert.Equal(t, expectedDirs, dirs)
}
