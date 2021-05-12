// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/kibana"
	log "github.com/sirupsen/logrus"
)

// Package represents the operations that can be performed by an installer package type
type Package interface {
	Install(containerName string, token string) error
	InstallCerts(cfg *kibana.FleetConfig) error
	PrintLogs(containerName string) error
	Postinstall() error
	Preinstall() error
	Uninstall() error
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

// extractPackage depends on the underlying OS, so 'cmds' must contain the specific instructions for the OS
func (i *BasePackage) extractPackage(cmds []string) error {
	sm := deploy.NewServiceManager()
	err := sm.ExecCommandInService(
		deploy.NewServiceRequest(i.profile), deploy.NewServiceRequest(i.image), i.service, cmds, common.ProfileEnv, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmds,
			"error":   err,
			"image":   i.image,
			"service": i.service,
		}).Error("Could not extract agent package in the box")

		return err
	}

	return nil
}

// Postinstall executes operations after installing a DEB package
func (i *BasePackage) Postinstall() error {
	err := SystemctlRun(i.profile, i.image, i.service, "enable")
	if err != nil {
		return err
	}
	return SystemctlRun(i.profile, i.image, i.service, "start")
}

// PrintLogs prints logs for the agent
func (i *BasePackage) PrintLogs(containerName string) error {
	err := i.resolveLogFile(containerName)
	if err != nil {
		return fmt.Errorf("Could not resolve log file: %v", err)
	}

	cmd := []string{
		"cat", i.logFile,
	}

	sm := deploy.NewServiceManager()
	err = sm.ExecCommandInService(
		deploy.NewServiceRequest(i.profile), deploy.NewServiceRequest(i.image), i.service, cmd, common.ProfileEnv, false)
	if err != nil {
		return err
	}

	return nil
}

// resolveLogFile retrieves the full path of the log file in the underlying Docker container
// calculating the hash commit if necessary
func (i *BasePackage) resolveLogFile(containerName string) error {
	if strings.Contains(i.logFile, "%s") {
		hash, err := getElasticAgentHash(containerName, i.commitFile)
		if err != nil {
			log.WithFields(log.Fields{
				"containerName": containerName,
				"error":         err,
			}).Error("Could not get agent hash in the container")

			return err
		}

		i.logFile = fmt.Sprintf(i.logFile, hash)
	}

	return nil
}

// getElasticAgentHash uses Elastic Agent's home dir to read the file with agent's build hash
// it will return the first six characters of the hash (short hash)
func getElasticAgentHash(containerName string, commitFile string) (string, error) {
	cmd := []string{
		"cat", commitFile,
	}

	fullHash, err := deploy.ExecCommandIntoContainer(context.Background(), deploy.NewServiceRequest(containerName), "root", cmd)
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

// SystemctlRun runs systemctl in profile or service
func SystemctlRun(profile string, image string, service string, command string) error {
	cmd := []string{"systemctl", command, common.ElasticAgentProcessName}
	sm := deploy.NewServiceManager()
	err := sm.ExecCommandInService(
		deploy.NewServiceRequest(profile), deploy.NewServiceRequest(image), service, cmd, common.ProfileEnv, false)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
			"service": service,
		}).Errorf("Could not %s the service", command)

		return err
	}

	log.WithFields(log.Fields{
		"command": cmd,
		"service": service,
	}).Trace("Systemctl executed")
	return nil
}
