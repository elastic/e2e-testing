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

const repoBranch = "master"
const repoDomain = "github.com"
const repoName = "Hello-World"
const repoRemote = "octocat"

func TestBuild(t *testing.T) {
	var repo = ProjectBuilder.
		WithBaseWorkspace(".").
		WithDomain(repoDomain).
		WithRemote(repoRemote).
		WithName(repoName).
		Build()

	assert.Equal(t, ".", repo.BaseWorkspace)
	assert.Equal(t, repoDomain, repo.Domain)
	assert.Equal(t, "", repo.Protocol)
	assert.Equal(t, repoBranch, repo.Branch)
	assert.Equal(t, repoRemote, repo.User)
	assert.Equal(t, "https://"+repoDomain+"/"+repoRemote+"/"+repoName, repo.GetURL())
}

func TestBuildWithGitProtocol(t *testing.T) {
	var repo = ProjectBuilder.
		WithBaseWorkspace(".").
		WithGitProtocol().
		WithDomain(repoDomain).
		WithRemote(repoRemote).
		WithName(repoName).
		Build()

	assert.Equal(t, ".", repo.BaseWorkspace)
	assert.Equal(t, repoDomain, repo.Domain)
	assert.Equal(t, "git@", repo.Protocol)
	assert.Equal(t, repoBranch, repo.Branch)
	assert.Equal(t, repoRemote, repo.User)
	assert.Equal(t, "git@"+repoDomain+":"+repoRemote+"/"+repoName, repo.GetURL())
}

func TestBuildWithWrongRemote(t *testing.T) {
	var repo = ProjectBuilder.
		WithBaseWorkspace(".").
		WithDomain(repoDomain).
		WithRemote(repoRemote).
		WithName(repoName).
		Build()

	assert.Equal(t, repoBranch, repo.Branch)
	assert.Equal(t, repoRemote, repo.User)
}

func TestBuildWithWellFormedRemote(t *testing.T) {
	var repo = ProjectBuilder.
		WithBaseWorkspace(".").
		WithDomain(repoDomain).
		WithRemote(repoRemote + ":foo").
		WithName(repoName).
		Build()

	assert.Equal(t, "foo", repo.Branch)
	assert.Equal(t, repoRemote, repo.User)
}

func TestClone(t *testing.T) {
	defer filet.CleanUp(t)
	gitDir := createGitDir(t)

	var repo = ProjectBuilder.
		WithBaseWorkspace(gitDir).
		WithDomain(repoDomain).
		WithRemote(repoRemote + ":" + repoBranch).
		WithName(repoName).
		Build()

	Clone(repo)

	e, _ := Exists(path.Join(gitDir, repoName))
	assert.True(t, e)
}

func createGitDir(t *testing.T) string {
	tmpDir := filet.TmpDir(t, "")
	gitDir := path.Join(tmpDir, "git")

	err := MkdirAll(gitDir)
	assert.Nil(t, err)

	return gitDir
}
