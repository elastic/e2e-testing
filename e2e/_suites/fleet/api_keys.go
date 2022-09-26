// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/cucumber/godog"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

func (fts *FleetTestSuite) getAgentDefaultAPIKey() (string, error) {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.getDeployer().GetServiceManifest(fts.currentContext, agentService)
	agent, err := fts.kibanaClient.GetAgentByHostnameFromList(fts.currentContext, manifest.Hostname)
	if err != nil {
		return "", err
	}
	return agent.DefaultAPIKey, nil
}

func (fts *FleetTestSuite) getAgentPermissionHashes() (map[string]string, error) {
	agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
	manifest, _ := fts.getDeployer().GetServiceManifest(fts.currentContext, agentService)
	agent, err := fts.kibanaClient.GetAgentByHostnameFromList(fts.currentContext, manifest.Hostname)
	if err != nil {
		return nil, err
	}

	permissions := make(map[string]string)
	for _, output := range agent.Outputs {
		permissions[output.APIKeyID] = output.PermissionsHash
	}

	return permissions, nil
}

func (fts *FleetTestSuite) theAgentGetDefaultAPIKey() error {
	defaultAPIKey, _ := fts.getAgentDefaultAPIKey()
	log.WithFields(log.Fields{
		"default_api_key": defaultAPIKey,
	}).Info("The Agent is installed with Default Api Key")
	fts.DefaultAPIKey = defaultAPIKey

	hashes, _ := fts.getAgentPermissionHashes()
	fts.PermissionHashes = hashes

	return nil
}

func (fts *FleetTestSuite) verifyPermissionHashStatus(status string) error {
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	checkHashFn := func() error {
		hashes, _ := fts.getAgentPermissionHashes()

		logFields := log.Fields{
			"retries":     retryCount,
			"elapsedTime": exp.GetElapsedTime(),
		}

		permissionHashChanged := len(fts.PermissionHashes) != len(hashes)
		if !permissionHashChanged {
			for oldHash, oldPerm := range fts.PermissionHashes {
				newPerm, found := hashes[oldHash]
				if !found || oldPerm != newPerm {
					permissionHashChanged = true
					break
				}
			}
		}

		if status == "changed" {
			if !permissionHashChanged {
				retryCount++
				log.WithFields(logFields).Warn("Integration added and Output API Key did not change yet")
				return fmt.Errorf("integration added and Output API Key did not change yet")
			}

			log.WithFields(logFields).Infof("Default API Key has %s when the Integration has been added", status)
			return nil
		}

		if status == "not changed" {
			if permissionHashChanged {
				retryCount++
				log.WithFields(logFields).Error("Integration updated and Output API Key is still changed")
				return fmt.Errorf("integration updated and Output API Key is still changed")
			}

			log.WithFields(logFields).Infof("Output API Key has %s when the Integration has been updated", status)
			return nil
		}

		log.Warnf("Status %s is not supported yet", status)
		return godog.ErrPending
	}

	err := backoff.Retry(checkHashFn, exp)
	return err
}

func (fts *FleetTestSuite) verifyDefaultAPIKey(status string) error {
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	retryCount := 1

	exp := utils.GetExponentialBackOff(maxTimeout)

	checkAPIKeyFn := func() error {
		newDefaultAPIKey, _ := fts.getAgentDefaultAPIKey()

		logFields := log.Fields{
			"new_default_api_key": newDefaultAPIKey,
			"old_default_api_key": fts.DefaultAPIKey,
			"retries":             retryCount,
			"elapsedTime":         exp.GetElapsedTime(),
		}

		defaultAPIKeyHasChanged := (newDefaultAPIKey != fts.DefaultAPIKey)

		if status == "changed" {
			if !defaultAPIKeyHasChanged {
				retryCount++
				log.WithFields(logFields).Warn("Integration added and Default API Key did not change yet")
				return fmt.Errorf("integration added and Default API Key did not change yet")
			}

			log.WithFields(logFields).Infof("Default API Key has %s when the Integration has been added", status)
			return nil
		}

		if status == "not changed" {
			if defaultAPIKeyHasChanged {
				retryCount++
				log.WithFields(logFields).Error("Integration updated and Default API Key is still changed")
				return fmt.Errorf("integration updated and Default API Key is still changed")
			}

			log.WithFields(logFields).Infof("Default API Key has %s when the Integration has been updated", status)
			return nil
		}

		log.Warnf("Status %s is not supported yet", status)
		return godog.ErrPending
	}

	err := backoff.Retry(checkAPIKeyFn, exp)
	return err
}
