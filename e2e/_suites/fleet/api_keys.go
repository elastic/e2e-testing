// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (fts *FleetTestSuite) getAgentDefaultAPIKey() (string, error) {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.getDeployer().Inspect(fts.currentContext, agentService)
	agent, err := fts.kibanaClient.GetAgentByHostnameFromList(fts.currentContext, manifest.Hostname)
	if err != nil {
		return "", err
	}
	return agent.DefaultAPIKey, nil
}

func (fts *FleetTestSuite) theAgentGetDefaultAPIKey() error {
	defaultAPIKey, _ := fts.getAgentDefaultAPIKey()
	log.WithFields(log.Fields{
		"default_api_key": defaultAPIKey,
	}).Info("The Agent is installed with Default Api Key")
	fts.DefaultAPIKey = defaultAPIKey
	return nil
}

func (fts *FleetTestSuite) verifyDefaultAPIKey(status string) error {
	newDefaultAPIKey, _ := fts.getAgentDefaultAPIKey()

	logFields := log.Fields{
		"new_default_api_key": newDefaultAPIKey,
		"old_default_api_key": fts.DefaultAPIKey,
	}

	defaultAPIKeyHasChanged := (newDefaultAPIKey != fts.DefaultAPIKey)

	if status == "changed" {
		if !defaultAPIKeyHasChanged {
			log.WithFields(logFields).Error("Integration added and Default API Key do not change")
			return errors.New("Integration added and Default API Key do not change")
		}

		log.WithFields(logFields).Infof("Default API Key has %s when the Integration has been added", status)
		return nil
	}

	if status == "not changed" {
		if defaultAPIKeyHasChanged {
			log.WithFields(logFields).Error("Integration updated and Default API Key is changed")
			return errors.New("Integration updated and Default API Key is changed")
		}

		log.WithFields(logFields).Infof("Default API Key has %s when the Integration has been updated", status)
		return nil
	}

	log.Warnf("Status %s is not supported yet", status)
	return godog.ErrPending
}
