// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"context"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	tc "github.com/testcontainers/testcontainers-go"
)

func Test_CopyFile(t *testing.T) {
	ctx := context.Background()

	containerName := "server"
	c, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image:      "busybox",
			Name:       containerName,
			Entrypoint: []string{"sleep", "300"},
		},
		Started: true,
	})
	assert.Nil(t, err)

	defer func() {
		err := c.Terminate(ctx)
		assert.Nil(t, err)
	}()

	t.Run("Copy file succeded", func(t *testing.T) {
		src := path.Join("..", "_testresources", "dockerCopy.txt")
		target := "/tmp"

		err = CopyFileToContainer(ctx, containerName, src, target, false)
		assert.Nil(t, err)

		output, err := ExecCommandIntoContainer(ctx, NewServiceRequest(containerName), "root", []string{"cat", "/tmp/dockerCopy.txt"})
		assert.Nil(t, err)
		assert.True(t, strings.HasSuffix(output, "OK!"), "File contains the 'OK!' string")
	})

	t.Run("Copy file raises error with invalid source path", func(t *testing.T) {
		src := path.Join("..", "this", "path", "does", "not", "exist", "dockerCopy.txt")
		target := "/tmp"

		err = CopyFileToContainer(ctx, containerName, src, target, false)
		assert.NotNil(t, err)
	})

	t.Run("Copy file raises error with invalid target parent dir", func(t *testing.T) {
		src := path.Join("..", "_testresources", "dockerCopy.txt")
		target := "/this-path-does-not-exist"

		err = CopyFileToContainer(ctx, containerName, src, target, false)
		assert.NotNil(t, err, "Parent path '/this-path-does-not-exist' should exist in the container")
	})

	t.Run("Copy file raises error with invalid target subdir", func(t *testing.T) {
		src := path.Join("..", "_testresources", "dockerCopy.txt")
		target := "/tmp/subdir/"

		err = CopyFileToContainer(ctx, containerName, src, target, false)
		assert.NotNil(t, err, "Parent path '/tmp/subdir' should not exist in the container")
	})

	t.Run("Copy tar file", func(t *testing.T) {
		src := path.Join("..", "_testresources", "sample.tar.gz")
		target := "/"

		err = CopyFileToContainer(ctx, containerName, src, target, true)
		assert.Nil(t, err)

		output, err := ExecCommandIntoContainer(ctx, NewServiceRequest(containerName), "root", []string{"ls", "/project/txtr/kermit.jpg"})
		assert.Nil(t, err)
		assert.True(t, strings.Contains(output, "/project/txtr/kermit.jpg"), "File '/project/txtr/kermit.jpg' should be present")
	})
}
