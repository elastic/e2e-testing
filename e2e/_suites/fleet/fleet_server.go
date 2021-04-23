// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

func (fts *FleetTestSuite) bootstrapFleetServerWithInstaller(image string, installerType string) error {
	fts.ElasticAgentStopped = true
	return fts.anAgentIsDeployedToFleetWithInstallerAndFleetServer(image, installerType, true)
}
