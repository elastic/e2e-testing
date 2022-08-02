// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/installer"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

func (fts *FleetTestSuite) anAttemptToEnrollANewAgentFails() error {
	log.Trace("Enrolling a new agent with an revoked token")

	serviceName := common.ElasticAgentServiceName
	agentService := deploy.NewServiceRequest(serviceName)
	agentInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, fts.InstallerType)
	err := agentInstaller.Uninstall(fts.currentContext)
	if err != nil {
		log.Errorf("could not uninstall the current agent: %v", err)
		return err
	}

	err = fts.deployAgentToFleet(InstallerType(fts.InstallerType))
	if err == nil {
		err = fmt.Errorf("the agent was enrolled although the token was previously revoked")

		log.WithFields(log.Fields{
			"tokenID": fts.CurrentTokenID,
			"error":   err,
		}).Error(err.Error())
		return err
	}

	// checking the error message produced by the install command in TAR installer
	// to distinguish from other install errors
	if err != nil && strings.Contains(err.Error(), "Error: enroll command failed") {
		log.WithFields(log.Fields{
			"err":   err,
			"token": fts.CurrentToken,
		}).Debug("As expected, it's not possible to enroll an agent with a revoked token")
		return nil
	}

	return nil
}

func (fts *FleetTestSuite) theAgentIsUnenrolled() error {
	return fts.unenrollHostname()
}

func (fts *FleetTestSuite) theAgentIsReenrolledOnTheHost() error {
	log.Trace("Re-enrolling the agent on the host with same token")

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	agentInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, fts.InstallerType)

	err := agentInstaller.Enroll(fts.currentContext, fts.CurrentToken, fts.ElasticAgentFlags)
	if err != nil {
		return err
	}

	return nil
}

func (fts *FleetTestSuite) theEnrollmentTokenIsRevoked() error {
	log.WithFields(log.Fields{
		"token":   fts.CurrentToken,
		"tokenID": fts.CurrentTokenID,
	}).Trace("Revoking enrollment token")

	err := fts.kibanaClient.DeleteEnrollmentAPIKey(fts.currentContext, fts.CurrentTokenID)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"token":   fts.CurrentToken,
		"tokenID": fts.CurrentTokenID,
	}).Debug("Token was revoked")

	// FIXME: Remove once https://github.com/elastic/kibana/issues/105078 is addressed
	utils.Sleep(time.Duration(utils.TimeoutFactor) * 20 * time.Second)
	return nil
}

// unenrollHostname deletes the statuses for an existing agent, filtering by hostname
func (fts *FleetTestSuite) unenrollHostname() error {
	span, _ := apm.StartSpanOptions(fts.currentContext, "Unenrolling hostname", "elastic-agent.hostname.unenroll", apm.SpanOptions{
		Parent: apm.SpanFromContext(fts.currentContext).TraceContext(),
	})
	defer span.End()

	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.getDeployer().GetServiceManifest(fts.currentContext, agentService)
	log.Tracef("Un-enrolling all agentIDs for %s", manifest.Hostname)

	agents, err := fts.kibanaClient.ListAgents(fts.currentContext)
	if err != nil {
		return err
	}

	for _, agent := range agents {
		if agent.LocalMetadata.Host.HostName == manifest.Hostname {
			log.WithFields(log.Fields{
				"hostname": manifest.Hostname,
			}).Debug("Un-enrolling agent in Fleet")

			err := fts.kibanaClient.UnEnrollAgent(fts.currentContext, agent.LocalMetadata.Host.HostName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
