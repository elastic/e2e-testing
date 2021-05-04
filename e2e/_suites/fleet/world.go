// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/docker"
)

// IngestManagerTestSuite represents a test suite, holding references to the pieces needed to run the tests
type IngestManagerTestSuite struct {
	Fleet *FleetTestSuite
}

func (imts *IngestManagerTestSuite) processStateOnTheHost(process string, state string) error {
	profile := common.FleetProfileName

	var containerName string

	if imts.Fleet.StandAlone {
		containerName = fmt.Sprintf("%s_%s_%d", profile, common.ElasticAgentServiceName, 1)
	} else {
		agentInstaller := imts.Fleet.getInstaller()
		containerName = imts.Fleet.getContainerName(agentInstaller, 1)
	}

	return docker.CheckProcessStateOnTheHost(containerName, process, state, common.TimeoutFactor)
}
