// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"runtime"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/installer"
	log "github.com/sirupsen/logrus"
)

// this step infers the installer type from the underlying OS image
// supported Docker images: centos and debian
func (fts *FleetTestSuite) anAgentIsDeployedToFleet(image string) error {
	installerType := "rpm"
	if image == "debian" {
		installerType = "deb"
	}
	fts.BeatsProcess = ""

	// FIXME: We need to cleanup the steps to support different operating systems
	// for now we will force the zip installer type when the agent is running on windows
	if runtime.GOOS == "windows" && common.Provider == "remote" {
		installerType = "zip"
	}
	return fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(installerType)
}

func (fts *FleetTestSuite) anAgentIsDeployedToFleetOnTopOfBeat(beatsProcess string) error {
	installerType := "tar"

	// FIXME: We need to cleanup the steps to support different operating systems
	// for now we will force the zip installer type when the agent is running on windows
	if runtime.GOOS == "windows" && common.Provider == "remote" {
		installerType = "zip"
	}

	fts.BeatsProcess = beatsProcess

	return fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(installerType)
}

// supported installers: tar, rpm, deb
func (fts *FleetTestSuite) anAgentIsDeployedToFleetWithInstaller(installerType string) error {
	fts.BeatsProcess = ""

	// FIXME: We need to cleanup the steps to support different operating systems
	// for now we will force the zip installer type when the agent is running on windows
	if runtime.GOOS == "windows" && common.Provider == "remote" {
		installerType = "zip"
	}

	return fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(installerType)
}

// supported installers: tar, rpm, deb
func (fts *FleetTestSuite) anAgentIsDeployedToFleetWithInstallerAndTags(installerType string, flags string) error {
	fts.BeatsProcess = ""

	// FIXME: We need to cleanup the steps to support different operating systems
	// for now we will force the zip installer type when the agent is running on windows
	if runtime.GOOS == "windows" && common.Provider == "remote" {
		installerType = "zip"
	}
	fts.ElasticAgentFlags = flags
	return fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(installerType)
}

func (fts *FleetTestSuite) anAgentIsDeployedToFleetWithInstallerAndFleetServer(installerType string) error {
	log.WithFields(log.Fields{
		"installer": installerType,
	}).Trace("Deploying an agent to Fleet with base image using an already bootstrapped Fleet Server")

	deployedAgentsCount++

	fts.InstallerType = installerType

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName).
		WithScale(deployedAgentsCount).
		WithVersion(fts.Version)

	if fts.BeatsProcess != "" {
		agentService = agentService.WithBackgroundProcess(fts.BeatsProcess)
	}

	services := []deploy.ServiceRequest{
		agentService,
	}
	env := fts.getProfileEnv()
	err := fts.getDeployer().Add(fts.currentContext, deploy.NewServiceRequest(common.FleetProfileName), services, env)
	if err != nil {
		return err
	}

	agentInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, installerType)
	err = deployAgentToFleet(fts.currentContext, agentInstaller, fts.CurrentToken, fts.ElasticAgentFlags)
	if err != nil {
		return err
	}
	return err
}

func deployAgentToFleet(ctx context.Context, agentInstaller deploy.ServiceOperator, token string, flags string) error {
	err := agentInstaller.Preinstall(ctx)
	if err != nil {
		return err
	}

	err = agentInstaller.Install(ctx)
	if err != nil {
		return err
	}

	err = agentInstaller.Enroll(ctx, token, flags)
	if err != nil {
		return err
	}

	return agentInstaller.Postinstall(ctx)
}
