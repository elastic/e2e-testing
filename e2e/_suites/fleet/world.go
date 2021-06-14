// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

// IngestManagerTestSuite represents a test suite, holding references to the pieces needed to run the tests
type IngestManagerTestSuite struct {
	Fleet *FleetTestSuite
}

func (imts *IngestManagerTestSuite) processStateOnTheHost(process string, state string) error {
	ocurrences := "1"
	if state == "uninstalled" || state == "stopped" {
		ocurrences = "0"
	}
	return imts.thereAreInstancesOfTheProcessInTheState(ocurrences, process, state)
}

func (imts *IngestManagerTestSuite) thereAreInstancesOfTheProcessInTheState(ocurrences string, process string, state string) error {
	profile := common.FleetProfileName

	var containerName string

	if imts.Fleet.StandAlone {
		containerName = fmt.Sprintf("%s_%s_%d", profile, common.ElasticAgentServiceName, 1)
	} else {
		agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
		manifest, _ := imts.Fleet.deployer.Inspect(imts.Fleet.currentContext, agentService)
		containerName = manifest.Name
	}

	count, err := strconv.Atoi(ocurrences)
	if err != nil {
		return err
	}

	return CheckProcessState(imts.Fleet.deployer, containerName, process, state, count, utils.TimeoutFactor)
}

// CheckProcessState checks if a process is in the desired state in a container
// name of the container for the service:
// we are using the underlying deployer to run the commands in the container/service
func CheckProcessState(deployer deploy.Deployment, service string, process string, state string, occurrences int, timeoutFactor int) error {
	timeout := time.Duration(utils.TimeoutFactor) * time.Minute

	err := waitForProcess(deployer, service, process, state, occurrences, timeout)
	if err != nil {
		if state == "started" {
			log.WithFields(log.Fields{
				"service ": service,
				"error":    err,
				"timeout":  timeout,
			}).Error("The process was not found but should be present")
		} else {
			log.WithFields(log.Fields{
				"service": service,
				"error":   err,
				"timeout": timeout,
			}).Error("The process was found but shouldn't be present")
		}

		return err
	}

	return nil
}

// waitForProcess polls a container executing "ps" command until the process is in the desired state (present or not),
// or a timeout happens
func waitForProcess(deployer deploy.Deployment, service string, process string, desiredState string, ocurrences int, maxTimeout time.Duration) error {
	exp := utils.GetExponentialBackOff(maxTimeout)

	mustBePresent := false
	if desiredState == "started" {
		mustBePresent = true
	}
	retryCount := 1

	// wrap service into a request for the deployer
	serviceRequest := deploy.NewServiceRequest(service)

	processStatus := func() error {
		log.WithFields(log.Fields{
			"desiredState": desiredState,
			"ocurrences":   ocurrences,
			"process":      process,
		}).Trace("Checking process desired state on the container")

		// pgrep -d: -d, --delimiter <string>  specify output delimiter
		//i.e. "pgrep -d , metricbeat": 483,519
		cmds := []string{"pgrep", "-d", ",", process}
		output, err := deployer.ExecIn(context.Background(), common.FleetProfileServiceRequest, serviceRequest, cmds)
		if err != nil {

			if !mustBePresent && ocurrences == 0 {
				log.WithFields(log.Fields{
					"cmds":          cmds,
					"desiredState":  desiredState,
					"elapsedTime":   exp.GetElapsedTime(),
					"error":         err,
					"service":       service,
					"mustBePresent": mustBePresent,
					"ocurrences":    ocurrences,
					"process":       process,
					"retry":         retryCount,
				}).Warn("Process is not present and number of occurences is 0")
				return nil
			}

			log.WithFields(log.Fields{
				"cmds":          cmds,
				"desiredState":  desiredState,
				"elapsedTime":   exp.GetElapsedTime(),
				"error":         err,
				"service":       service,
				"mustBePresent": mustBePresent,
				"ocurrences":    ocurrences,
				"process":       process,
				"retry":         retryCount,
			}).Warn("Could not get number of processes in the container")

			retryCount++

			return err
		}

		// tokenize the pids to get each pid's state, adding them to an array if they match the desired state
		// From Split docs:
		// If output does not contain sep and sep is not empty, Split returns a
		// slice of length 1 whose only element is s, that's why we first initialise to the empty array
		pids := strings.Split(output, ",")
		if len(pids) == 1 && pids[0] == "" {
			pids = []string{}
		}

		log.WithFields(log.Fields{
			"count":         len(pids),
			"desiredState":  desiredState,
			"mustBePresent": mustBePresent,
			"pids":          pids,
			"process":       process,
		}).Tracef("Pids for process found")

		desiredStatePids := []string{}

		for _, pid := range pids {
			pidStateCmds := []string{"ps", "-q", pid, "-o", "state", "--no-headers"}
			pidState, err := deployer.ExecIn(context.Background(), common.FleetProfileServiceRequest, serviceRequest, pidStateCmds)
			if err != nil {
				log.WithFields(log.Fields{
					"cmds":          cmds,
					"desiredState":  desiredState,
					"elapsedTime":   exp.GetElapsedTime(),
					"error":         err,
					"service":       service,
					"mustBePresent": mustBePresent,
					"ocurrences":    ocurrences,
					"pid":           pid,
					"process":       process,
					"retry":         retryCount,
				}).Warn("Could not check pid status in the container")

				retryCount++

				return err
			}

			log.WithFields(log.Fields{
				"desiredState":  desiredState,
				"mustBePresent": mustBePresent,
				"pid":           pid,
				"pidState":      pidState,
				"process":       process,
			}).Tracef("Checking if process is in the S state")

			// if the process must be present, then check for the S state
			// From 'man ps':
			// D    uninterruptible sleep (usually IO)
			// R    running or runnable (on run queue)
			// S    interruptible sleep (waiting for an event to complete)
			// T    stopped by job control signal
			// t    stopped by debugger during the tracing
			// W    paging (not valid since the 2.6.xx kernel)
			// X    dead (should never be seen)
			// Z    defunct ("zombie") process, terminated but not reaped by its parent
			if mustBePresent && pidState == "S" {
				desiredStatePids = append(desiredStatePids, pid)
			} else if !mustBePresent {
				desiredStatePids = append(desiredStatePids, pid)
			}
		}

		occurrencesMatched := (len(desiredStatePids) == ocurrences)

		// both true or both false
		if mustBePresent == occurrencesMatched {
			log.WithFields(log.Fields{
				"desiredOcurrences": ocurrences,
				"desiredState":      desiredState,
				"service":           service,
				"mustBePresent":     mustBePresent,
				"ocurrences":        len(desiredStatePids),
				"process":           process,
			}).Infof("Process desired state checked")

			return nil
		}

		if mustBePresent {
			err = fmt.Errorf("%s process is not running in the container with the desired number of occurrences (%d) yet", process, ocurrences)
			log.WithFields(log.Fields{
				"desiredOcurrences": ocurrences,
				"desiredState":      desiredState,
				"elapsedTime":       exp.GetElapsedTime(),
				"error":             err,
				"service":           service,
				"ocurrences":        len(desiredStatePids),
				"process":           process,
				"retry":             retryCount,
			}).Warn(err.Error())

			retryCount++

			return err
		}

		err = fmt.Errorf("%s process is still running in the container", process)
		log.WithFields(log.Fields{
			"desiredOcurrences": ocurrences,
			"elapsedTime":       exp.GetElapsedTime(),
			"error":             err,
			"service":           service,
			"ocurrences":        len(desiredStatePids),
			"process":           process,
			"state":             desiredState,
			"retry":             retryCount,
		}).Warn(err.Error())

		retryCount++

		return err
	}

	err := backoff.Retry(processStatus, exp)
	if err != nil {
		return err
	}

	return nil
}
