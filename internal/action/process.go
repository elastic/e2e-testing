// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package action

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

// actionWaitProcess implements operation for waiting on a process status
type actionWaitProcess struct {
	service deploy.ServiceRequest
	deploy  deploy.Deployment
	opts    map[string]string
}

// AttachActionWaitProcess action to woit for a process status on *nix like systems
func AttachActionWaitProcess(deploy deploy.Deployment, service deploy.ServiceRequest, actionOpts map[string]string) deploy.ServiceOperatorAction {
	return &actionWaitProcess{
		service: service,
		deploy:  deploy,
		opts:    actionOpts,
	}
}

// Run executes the command
func (a *actionWaitProcess) Run(ctx context.Context) (string, error) {
	timeoutFactor, _ := time.ParseDuration(a.opts["maxTimeout"])
	exp := utils.GetExponentialBackOff(timeoutFactor)

	mustBePresent := false
	if a.opts["desiredState"] == "started" {
		mustBePresent = true
	}
	retryCount := 1

	processStatus := func() error {
		occurrences, _ := strconv.Atoi(a.opts["occurrences"])
		log.WithFields(log.Fields{
			"desiredState": a.opts["desiredState"],
			"occurrences":  occurrences,
			"process":      a.opts["process"],
		}).Trace("Checking process desired state on the container")

		// pgrep -d: -d, --delimiter <string>  specify output delimiter
		//i.e. "pgrep -d , metricbeat": 483,519
		cmds := []string{"pgrep", "-d", ",", a.opts["process"]}
		output, err := a.deploy.ExecIn(ctx, common.FleetProfileServiceRequest, a.service, cmds)
		if err != nil {
			if !mustBePresent && occurrences == 0 {
				log.WithFields(log.Fields{
					"cmds":          cmds,
					"desiredState":  a.opts["desiredState"],
					"elapsedTime":   exp.GetElapsedTime(),
					"error":         err,
					"service":       a.service,
					"mustBePresent": mustBePresent,
					"occurrences":   a.opts["occurrences"],
					"process":       a.opts["process"],
					"retry":         retryCount,
				}).Warn("Process is not present and number of occurences is 0")
				return nil
			}

			log.WithFields(log.Fields{
				"cmds":          cmds,
				"desiredState":  a.opts["desiredState"],
				"elapsedTime":   exp.GetElapsedTime(),
				"error":         err,
				"service":       a.service,
				"mustBePresent": mustBePresent,
				"occurrences":   a.opts["occurrences"],
				"process":       a.opts["process"],
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
			"desiredState":  a.opts["desiredState"],
			"mustBePresent": mustBePresent,
			"pids":          pids,
			"process":       a.opts["process"],
		}).Tracef("Pids for process found")

		desiredStatePids := []string{}

		for _, pid := range pids {
			pidStateCmds := []string{"ps", "-q", pid, "-o", "state", "--no-headers"}
			pidState, err := a.deploy.ExecIn(ctx, common.FleetProfileServiceRequest, a.service, pidStateCmds)
			if err != nil {
				log.WithFields(log.Fields{
					"cmds":          cmds,
					"desiredState":  a.opts["desiredState"],
					"elapsedTime":   exp.GetElapsedTime(),
					"error":         err,
					"service":       a.service,
					"mustBePresent": mustBePresent,
					"occurrences":   a.opts["occurrences"],
					"pid":           pid,
					"process":       a.opts["process"],
					"retry":         retryCount,
				}).Warn("Could not check pid status in the container")

				retryCount++

				return err
			}

			log.WithFields(log.Fields{
				"desiredState":  a.opts["desiredState"],
				"mustBePresent": mustBePresent,
				"pid":           pid,
				"pidState":      pidState,
				"process":       a.opts["process"],
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

		occurrencesMatched := (len(desiredStatePids) == occurrences)

		// both true or both false
		if mustBePresent == occurrencesMatched {
			log.WithFields(log.Fields{
				"desiredOccurrences": occurrences,
				"desiredState":       a.opts["desiredState"],
				"service":            a.service,
				"mustBePresent":      mustBePresent,
				"occurrences":        len(desiredStatePids),
				"process":            a.opts["process"],
			}).Infof("Process desired state checkedz")

			return nil
		}

		if mustBePresent {
			err = fmt.Errorf("%s process is not running in the container with the desired number of occurrences (%d) yet", a.opts["process"], occurrences)
			log.WithFields(log.Fields{
				"desiredOccurrences": occurrences,
				"desiredState":       a.opts["desiredState"],
				"elapsedTime":        exp.GetElapsedTime(),
				"error":              err,
				"service":            a.service,
				"occurrences":        len(desiredStatePids),
				"process":            a.opts["process"],
				"retry":              retryCount,
			}).Warn(err.Error())

			retryCount++

			return err
		}

		err = fmt.Errorf("%s process is still running in the container", a.opts["process"])
		log.WithFields(log.Fields{
			"desiredOccurrences": occurrences,
			"elapsedTime":        exp.GetElapsedTime(),
			"error":              err,
			"service":            a.service,
			"occurrences":        len(desiredStatePids),
			"process":            a.opts["process"],
			"state":              a.opts["desiredState"],
			"retry":              retryCount,
		}).Warn(err.Error())

		retryCount++

		return err
	}

	err := backoff.Retry(processStatus, exp)
	if err != nil {
		return "", err
	}

	return "", nil
}

// actionWaitProcessWin implements operation for waiting on a process on Windows
type actionWaitProcessWin struct {
	service deploy.ServiceRequest
	deploy  deploy.Deployment
	opts    map[string]string
}

// AttachActionWaitProcessWin action to wait for process status on windows systems
func AttachActionWaitProcessWin(deploy deploy.Deployment, service deploy.ServiceRequest, actionOpts map[string]string) deploy.ServiceOperatorAction {
	return &actionWaitProcessWin{
		service: service,
		deploy:  deploy,
		opts:    actionOpts,
	}
}

// Run executes the command
func (a *actionWaitProcessWin) Run(ctx context.Context) (string, error) {
	return "", nil
}
