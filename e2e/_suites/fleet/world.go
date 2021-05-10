// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"strconv"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/docker"
)

// IngestManagerTestSuite represents a test suite, holding references to the pieces needed to run the tests
type IngestManagerTestSuite struct {
	Fleet      *FleetTestSuite
	StandAlone *StandAloneTestSuite
}

func (imts *IngestManagerTestSuite) processStateOnTheHost(process string, state string) error {
	return imts.thereAreInstancesOfTheProcessInTheState("1", process, state)
}

func (imts *IngestManagerTestSuite) thereAreInstancesOfTheProcessInTheState(ocurrences string, process string, state string) error {
	profile := common.FleetProfileName
	serviceName := common.ElasticAgentServiceName

	containerName := fmt.Sprintf("%s_%s_%s_%d", profile, imts.Fleet.Image+"-systemd", serviceName, 1)
	if imts.StandAlone.Hostname != "" {
		containerName = fmt.Sprintf("%s_%s_%d", profile, serviceName, 1)
	}

	count, err := strconv.Atoi(ocurrences)
	if err != nil {
		return err
	}

	return docker.CheckProcessStateOnTheHost(containerName, process, state, count, common.TimeoutFactor)
}
