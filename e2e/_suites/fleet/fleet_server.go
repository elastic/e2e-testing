// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import log "github.com/sirupsen/logrus"

func (fts *FleetTestSuite) bootstrapFleetServerWithInstaller(image string, installerType string) error {
	fts.ElasticAgentStopped = true

	log.WithFields(log.Fields{
		"image":     image,
		"installer": installerType,
	}).Trace("Bootstrapping fleet server for the agent")

	err := fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(image, installerType, true)
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"image":     image,
			"installer": installerType,
		}).Error("Fleet server could not be bootstrapped for the agent")
		return err
	}

	log.WithFields(log.Fields{
		"fleetServerHostname": fts.FleetServerHostname,
		"image":               image,
		"installer":           installerType,
	}).Info("Fleet server was bootstrapped for the agent")

	err = fts.theAgentIsListedInFleetWithStatus("online")
	if err != nil {
		log.WithFields(log.Fields{
			"error":               err,
			"fleetServerHostname": fts.FleetServerHostname,
			"image":               image,
			"installer":           installerType,
		}).Error("Fleet server could not reach the online status")
		return err
	}

	// the new compose files for fleet-server (centos/debian) are setting the hostname
	// we need it here, before getting the installer, to get the installer using fleet-server host
	fts.FleetServerHostname = "fleet-server-" + image

	return nil
}
