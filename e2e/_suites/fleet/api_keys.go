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
			"old_hashes":  fts.PermissionHashes,
			"new_hashes":  hashes,
			"retries":     retryCount,
			"elapsedTime": exp.GetElapsedTime(),
		}

		permissionHashChanged := len(fts.PermissionHashes) != len(hashes)
		permissionHashUpdated := false
		for oldHash, oldPerm := range fts.PermissionHashes {
			newPerm, found := hashes[oldHash]
			if !found {
				permissionHashChanged = true
			} else if oldPerm != newPerm {
				permissionHashUpdated = true
			}
		}

		if status == "changed" {
			if !permissionHashChanged {
				retryCount++
				log.WithFields(logFields).Warn("Integration added and Output API Key did not change yet")
				return fmt.Errorf("integration added and Output API Key did not change yet")
			}

			log.WithFields(logFields).Infof("Output API Key has %s when the Integration has been added", status)
			return nil
		}

		if status == "been updated" {
			if !permissionHashUpdated {
				retryCount++
				log.WithFields(logFields).Warn("Integration added and Output API Key did not updated yet")
				return fmt.Errorf("integration added and Output API Key did not updated yet")
			}

			log.WithFields(logFields).Infof("Output API Key has %s when the Integration has been added", status)
			return nil
		}

		if status == "not changed" {
			if permissionHashChanged || permissionHashUpdated {
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
