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

var deployedAgentsCount = 0

// this step infers the installer type from the underlying OS image
// supported Docker images: centos and debian
func (fts *FleetTestSuite) anAgentIsDeployedToFleet(image string) error {
	installerType := "rpm"
	if image == "debian" {
		installerType = "deb"
	}

	return fts.deployAgentToFleet(InstallerType(installerType))
}

func (fts *FleetTestSuite) anAgentIsDeployedToFleetOnTopOfBeat(beatsProcess string) error {
	return fts.deployAgentToFleet(InstallerType("tar"), BeatsProcess(beatsProcess))
}

// supported installers: tar, rpm, deb
func (fts *FleetTestSuite) anAgentIsDeployedToFleetWithInstaller(installerType string) error {
	return fts.deployAgentToFleet(InstallerType(installerType))
}

// supported installers: tar, rpm, deb
func (fts *FleetTestSuite) anAgentIsDeployedToFleetWithInstallerAndTags(installerType string, flags string) error {
	return fts.deployAgentToFleet(InstallerType(installerType), Flags(flags))
}

func (fts *FleetTestSuite) deployAgentToFleet(opts ...DeploymentOpt) error {
	// Default Options
	args := &DeploymentOpts{
		beatsProcess:        "",
		installerType:       "tar",
		flags:               "",
		boostrapFleetServer: false,
	}

	for _, opt := range opts {
		opt(args)
		log.Info("<<< configuration to agent deployment applied")
	}

	log.WithFields(log.Fields{
		"installer": args.installerType,
	}).Trace("Deploying an agent to Fleet with base image using an already bootstrapped Fleet Server")

	deployedAgentsCount++

	fts.InstallerType = args.installerType
	fts.BeatsProcess = args.beatsProcess
	fts.ElasticAgentFlags = args.flags

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

	agentInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, fts.InstallerType)
	err = deploymentLifecycle(fts.currentContext, agentInstaller, fts.CurrentToken, fts.ElasticAgentFlags)
	if err != nil {
		return err
	}
	return err
}

// DeploymentOpts options to be applied to a deployment of the elastic-agent
type DeploymentOpts struct {
	beatsProcess        string
	boostrapFleetServer bool
	installerType       string
	flags               string
}

// DeploymentOpt an option to be applied to a deployment of the elastic-agent
type DeploymentOpt func(*DeploymentOpts)

// BeatsProcess option to start a Beats process before the agent is running. Default is empty
func BeatsProcess(beatsProcess string) DeploymentOpt {
	return func(args *DeploymentOpts) {
		log.Tracef(">>> applying configuration to agent deployment [BeatsProcess]: %s", beatsProcess)
		args.beatsProcess = beatsProcess
	}
}

// BootstrapFleetServer option to bootstrap a Fleet Server, otherwise the stack one will be used. Default is false
func BootstrapFleetServer(boostrapFleetServer bool) DeploymentOpt {
	return func(args *DeploymentOpts) {
		log.Tracef(">>> applying configuration to agent deployment [BootstrapFleetServer]: %t", boostrapFleetServer)
		args.boostrapFleetServer = boostrapFleetServer
	}
}

// InstallerType option to define the installer to use for the agent. Default is "tar"
func InstallerType(installerType string) DeploymentOpt {
	// FIXME: We need to cleanup the steps to support different operating systems
	// for now we will force the zip installer type when the agent is running on windows
	if runtime.GOOS == "windows" && common.Provider == "remote" {
		installerType = "zip"
	}

	return func(args *DeploymentOpts) {
		log.Tracef(">>> applying configuration to agent deployment [InstallerType]: %s", installerType)
		args.installerType = installerType
	}
}

// Flags option to pass flags to the enrollment of the agent. Default is empty
func Flags(flags string) DeploymentOpt {
	return func(args *DeploymentOpts) {
		log.Tracef(">>> applying configuration to agent deployment [Flags]: %s", flags)
		args.flags = flags
	}
}

func deploymentLifecycle(ctx context.Context, agentInstaller deploy.ServiceOperator, token string, flags string) error {
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
