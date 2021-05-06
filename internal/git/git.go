// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package git

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lann/builder"
	log "github.com/sirupsen/logrus"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
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
func Clone(repositories ...Project) error {
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
			return err
		}
	}
	return nil
}

func cloneGithubRepository(githubRepo Project, resultChannel chan bool, errorChannel chan error) {

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
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(githubRepo.Branch),
		Depth:         1,
		Auth:          auth(githubRepo.Protocol),
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

// Fetch performs a fetch and checkout of the specified remote branch
func Fetch(beats Project) error {
	repo, err := git.PlainOpen(beats.GetWorkspace())
	if err != nil {
		return err
	}

	fetchOptions := &git.FetchOptions{
		RemoteName: "origin",
		Depth:      1,
		Progress:   os.Stdout,
		Force:      true,
		Auth:       auth(beats.Protocol),
	}

	err = repo.Fetch(fetchOptions)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}

	tree, err := repo.Worktree()
	if err != nil {
		return err
	}

	ref := plumbing.NewRemoteReferenceName("origin", beats.Branch)
	err = tree.Checkout(&git.CheckoutOptions{
		Branch: ref,
		Force:  true,
	})
	return err
}

func auth(protocol string) transport.AuthMethod {
	if protocol == GitProtocol {
		auth, err := ssh.NewSSHAgentAuth("git")
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Cloning using keys from SSH agent failed")
		}
		return auth
	}
	return nil
}
