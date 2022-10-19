// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"time"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"

	log "github.com/sirupsen/logrus"
)

// FleetTestSuite represents the scenarios for Fleet-mode
type FleetTestSuite struct {
	// integrations
	KibanaProfile       string
	StandAlone          bool
	CurrentToken        string // current enrollment token
	CurrentTokenID      string // current enrollment tokenID
	ElasticAgentStopped bool   // will be used to signal when the agent process can be called again in the tear-down stage
	Image               string // base image used to install the agent
	InstallerType       string
	Integration         kibana.IntegrationPackage // the installed integration
	Policy              kibana.Policy
	PolicyUpdatedAt     string // the moment the policy was updated
	Version             string // current elastic-agent version
	kibanaClient        *kibana.Client
	deployer            deploy.Deployment
	dockerDeployer      deploy.Deployment // used for docker related deployents, such as the stand-alone containers
	BeatsProcess        string            // (optional) name of the Beats that must be present before installing the elastic-agent
	// date controls for queries
	AgentStoppedDate             time.Time
	RuntimeDependenciesStartDate time.Time
	// instrumentation
	currentContext    context.Context
	DefaultAPIKey     string
	PermissionHashes  map[string]string
	ElasticAgentFlags string
}

func (fts *FleetTestSuite) getDeployer() deploy.Deployment {
	if fts.StandAlone {
		return fts.dockerDeployer
	}
	return fts.deployer
}

func (fts *FleetTestSuite) getProfileEnv() map[string]string {
	env := map[string]string{}

	for k, v := range common.ProfileEnv {
		env[k] = v
	}

	if fts.KibanaProfile != "" {
		env["kibanaProfile"] = fts.KibanaProfile
	}

	return env
}

func (fts *FleetTestSuite) theHostIsRestarted() error {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	err := fts.getDeployer().Stop(fts.currentContext, agentService)
	if err != nil {
		log.WithField("err", err).Error("Could not stop the service")
	}

	utils.Sleep(time.Duration(utils.TimeoutFactor) * 10 * time.Second)

	err = fts.getDeployer().Start(fts.currentContext, agentService)
	if err != nil {
		log.WithField("err", err).Error("Could not start the service")
	}

	log.Debug("The elastic-agent service has been restarted")
	return nil
}
