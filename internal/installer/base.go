// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"context"
	"strings"

	"github.com/elastic/e2e-testing/internal/deploy"
	log "github.com/sirupsen/logrus"
)

<<<<<<< HEAD
// Package represents the operations that can be performed by an installer package type
type Package interface {
	Install(containerName string, token string) error
	PrintLogs(containerName string) error
	Postinstall() error
	Preinstall() error
	Uninstall() error
=======
// Attach will attach a installer to a deployment allowing
// the installation of a package to be transparently configured no matter the backend
func Attach(deploy deploy.Deployment, service deploy.ServiceRequest, installType string) (deploy.ServiceOperator, error) {
	log.WithFields(log.Fields{
		"service":     service,
		"installType": installType,
	}).Trace("Attaching service for configuration")

	if strings.EqualFold(service.Name, "elastic-agent") {
		switch installType {
		case "tar":
			install := AttachElasticAgentTARPackage(deploy, service)
			return install, nil
		case "rpm":
			install := AttachElasticAgentRPMPackage(deploy, service)
			return install, nil
		case "deb":
			install := AttachElasticAgentDEBPackage(deploy, service)
			return install, nil
		case "docker":
			install := AttachElasticAgentDockerPackage(deploy, service)
			return install, nil
		}
	}

	return nil, nil
>>>>>>> f043003 (Feat installer rework 2 (#1208))
}

// BasePackage holds references to basic state for all installers
type BasePackage struct {
	binaryName string
	commitFile string
	image      string
	logFile    string
	profile    string
	service    string
}

// getElasticAgentHash uses Elastic Agent's home dir to read the file with agent's build hash
// it will return the first six characters of the hash (short hash)
func getElasticAgentHash(containerName string, commitFile string) (string, error) {
	cmd := []string{
		"cat", commitFile,
	}

	fullHash, err := deploy.ExecCommandIntoContainer(context.Background(), containerName, "root", cmd)
	if err != nil {
		return "", err
	}

	runes := []rune(fullHash)
	shortHash := string(runes[0:6])

	log.WithFields(log.Fields{
		"commitFile":    commitFile,
		"containerName": containerName,
		"hash":          fullHash,
		"shortHash":     shortHash,
	}).Debug("Agent build hash found")

	return shortHash, nil
}
