// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lann/builder"
	log "github.com/sirupsen/logrus"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	ssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

// GitProtocol the git protocol string representation
const GitProtocol = "git@"

// Project representes a git project
type Project struct {
	BaseWorkspace string
	Branch        string
	Domain        string
	Name          string
	Protocol      string
	User          string
}

// GetURL Returns the workspace of a Project
func (d *Project) GetURL() string {
	if d.Protocol == GitProtocol {
		return d.Protocol + d.Domain + ":" + d.User + "/" + d.Name
	}

	return "https://" + d.Domain + "/" + d.User + "/" + d.Name
}

// GetWorkspace Returns the workspace of a Project
func (d *Project) GetWorkspace() string {
	return filepath.Join(d.BaseWorkspace, d.Name)
}

type projectBuilder builder.Builder

func (b projectBuilder) Build() Project {
	return builder.GetStruct(b).(Project)
}

func (b projectBuilder) WithBaseWorkspace(baseWorkspace string) projectBuilder {
	return builder.Set(b, "BaseWorkspace", baseWorkspace).(projectBuilder)
}

func (b projectBuilder) WithDomain(domain string) projectBuilder {
	return builder.Set(b, "Domain", domain).(projectBuilder)
}

func (b projectBuilder) WithGitProtocol() projectBuilder {
	return builder.Set(b, "Protocol", GitProtocol).(projectBuilder)
}

func (b projectBuilder) WithName(name string) projectBuilder {
	return builder.Set(b, "Name", name).(projectBuilder)
}

func (b projectBuilder) WithRemote(remote string) projectBuilder {
	coordinates := strings.Split(remote, ":")
	if len(coordinates) == 1 {
		return b.withUser(coordinates[0]).withBranch("master")
	} else if len(coordinates) != 2 {
		return b
	}

	return b.withUser(coordinates[0]).withBranch(coordinates[1])
}

func (b projectBuilder) withBranch(branch string) projectBuilder {
	return builder.Set(b, "Branch", branch).(projectBuilder)
}

func (b projectBuilder) withUser(user string) projectBuilder {
	return builder.Set(b, "User", user).(projectBuilder)
}

// ProjectBuilder builder for git projects
var ProjectBuilder = builder.Register(projectBuilder{}, Project{}).(projectBuilder)

// Clone allows cloning an array of repositories simultaneously
func Clone(repositories ...Project) {
	repositoriesChannel := make(chan Project, len(repositories))
	for i := range repositories {
		repositoriesChannel <- repositories[i]
	}
	close(repositoriesChannel)

	workers := 5
	if len(repositoriesChannel) < workers {
		workers = len(repositoriesChannel)
	}

	errorChannel := make(chan error, 1)
	resultChannel := make(chan bool, len(repositories))

	for i := 0; i < workers; i++ {
		// Consume work from repositoriesChannel. Loop will end when no more work.
		for repository := range repositoriesChannel {
			go cloneGithubRepository(repository, resultChannel, errorChannel)
		}
	}

	// Collect results from workers

	for i := 0; i < len(repositories); i++ {
		select {
		case <-resultChannel:
			log.WithFields(log.Fields{
				"url": repositories[i].GetURL(),
			}).Info("Git clone succeed")
		case err := <-errorChannel:
			if err != nil {
				log.WithFields(log.Fields{
					"url":   repositories[i].GetURL(),
					"error": err,
				}).Warn("Git clone errored")
			}
		}
	}
}

func cloneGithubRepository(
	githubRepo Project, resultChannel chan bool, errorChannel chan error) {

	gitRepositoryDir := githubRepo.GetWorkspace()

	if _, err := os.Stat(gitRepositoryDir); os.IsExist(err) {
		select {
		case errorChannel <- err:
			// will break parent goroutine out of loop
		default:
			// don't care, first error wins
		}
		return
	}

	githubRepositoryURL := githubRepo.GetURL()

	log.WithFields(log.Fields{
		"url":       githubRepositoryURL,
		"directory": gitRepositoryDir,
	}).Info("Cloning project. This process could take long depending on its size")

	cloneOptions := &git.CloneOptions{
		URL:           githubRepositoryURL,
		Progress:      os.Stdout,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", githubRepo.Branch)),
		SingleBranch:  true,
	}

	if githubRepo.Protocol == GitProtocol {
		auth, err1 := ssh.NewSSHAgentAuth("git")
		if err1 != nil {
			log.WithFields(log.Fields{
				"error": err1,
			}).Fatal("Cloning using keys from SSH agent failed")
		}

		cloneOptions.Auth = auth
	}

	_, err := git.PlainClone(gitRepositoryDir, false, cloneOptions)

	if err != nil {
		select {
		case errorChannel <- err:
			// will break parent goroutine out of loop
		default:
			// don't care, first error wins
		}
		return
	}

	resultChannel <- true
}
