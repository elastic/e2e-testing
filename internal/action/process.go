// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package action

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

// ProcessAction contains the necessary options to pass into process action
type ProcessAction struct {
	Process      string
	DesiredState string
	Occurrences  int
	MaxTimeout   time.Duration
}

// actionWaitProcess implements operation for waiting on a process status
type actionWaitProcess struct {
	service deploy.ServiceRequest
	deploy  deploy.Deployment
	opts    ProcessAction
}

// AttachActionWaitProcess action to woit for a process status on *nix like systems
func AttachActionWaitProcess(deploy deploy.Deployment, service deploy.ServiceRequest, actionOpts ProcessAction) deploy.ServiceOperation {
	return &actionWaitProcess{
		service: service,
		deploy:  deploy,
		opts:    actionOpts,
	}
}

// Run executes the command
func (a *actionWaitProcess) Run(ctx context.Context) (string, error) {
	exp := utils.GetExponentialBackOff(a.opts.MaxTimeout)

	mustBePresent := false
	if a.opts.DesiredState == "started" {
		mustBePresent = true
	}
	retryCount := 1

	processStatus := func() error {
		log.WithFields(log.Fields{
			"desiredState": a.opts.DesiredState,
			"occurrences":  a.opts.Occurrences,
			"process":      a.opts.Process,
		}).Trace("Checking process desired state on the container")

		// pgrep -d: -d, --delimiter <string>  specify output delimiter
		//i.e. "pgrep -d , metricbeat": 483,519
		cmds := []string{"pgrep", "-d", ",", a.opts.Process}
		output, err := a.deploy.ExecIn(ctx, deploy.NewServiceRequest(common.FleetProfileName), a.service, cmds)
		if err != nil {
			if !mustBePresent && a.opts.Occurrences == 0 {
				log.WithFields(log.Fields{
					"cmds":          cmds,
					"desiredState":  a.opts.DesiredState,
					"elapsedTime":   exp.GetElapsedTime(),
					"error":         err,
					"service":       a.service,
					"mustBePresent": mustBePresent,
					"occurrences":   a.opts.Occurrences,
					"process":       a.opts.Process,
					"retry":         retryCount,
				}).Warn("Process is not present and number of occurences is 0")
				return nil
			}

			log.WithFields(log.Fields{
				"cmds":          cmds,
				"desiredState":  a.opts.DesiredState,
				"elapsedTime":   exp.GetElapsedTime(),
				"error":         err,
				"service":       a.service,
				"mustBePresent": mustBePresent,
				"occurrences":   a.opts.Occurrences,
				"process":       a.opts.Process,
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
			"desiredState":  a.opts.DesiredState,
			"mustBePresent": mustBePresent,
			"pids":          pids,
			"process":       a.opts.Process,
		}).Tracef("Pids for process found")

		desiredStatePids := []string{}

		for _, pid := range pids {
			pidStateCmds := []string{"ps", "-q", pid, "-o", "state", "--no-headers"}
			pidState, err := a.deploy.ExecIn(ctx, deploy.NewServiceRequest(common.FleetProfileName), a.service, pidStateCmds)
			if err != nil {
				log.WithFields(log.Fields{
					"cmds":          cmds,
					"desiredState":  a.opts.DesiredState,
					"elapsedTime":   exp.GetElapsedTime(),
					"error":         err,
					"service":       a.service,
					"mustBePresent": mustBePresent,
					"occurrences":   a.opts.Occurrences,
					"pid":           pid,
					"process":       a.opts.Process,
					"retry":         retryCount,
				}).Warn("Could not check pid status in the container")

				retryCount++

				return err
			}

			log.WithFields(log.Fields{
				"desiredState":  a.opts.DesiredState,
				"mustBePresent": mustBePresent,
				"pid":           pid,
				"pidState":      pidState,
				"process":       a.opts.Process,
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

		occurrencesMatched := (len(desiredStatePids) == a.opts.Occurrences)

		// both true or both false
		if mustBePresent == occurrencesMatched {
			log.WithFields(log.Fields{
				"desiredOccurrences": a.opts.Occurrences,
				"desiredState":       a.opts.DesiredState,
				"service":            a.service,
				"mustBePresent":      mustBePresent,
				"occurrences":        len(desiredStatePids),
				"process":            a.opts.Process,
			}).Infof("Process desired state checked")

			return nil
		}

		if mustBePresent {
			err = fmt.Errorf("%s process is not running in the container with the desired number of occurrences (%d) yet", a.opts.Process, a.opts.Occurrences)
			log.WithFields(log.Fields{
				"desiredOccurrences": a.opts.Occurrences,
				"desiredState":       a.opts.DesiredState,
				"elapsedTime":        exp.GetElapsedTime(),
				"error":              err,
				"service":            a.service,
				"occurrences":        len(desiredStatePids),
				"process":            a.opts.Process,
				"retry":              retryCount,
			}).Warn(err.Error())

			retryCount++

			return err
		}

		err = fmt.Errorf("%s process is still running in the container", a.opts.Process)
		log.WithFields(log.Fields{
			"desiredOccurrences": a.opts.Occurrences,
			"elapsedTime":        exp.GetElapsedTime(),
			"error":              err,
			"service":            a.service,
			"occurrences":        len(desiredStatePids),
			"process":            a.opts.Process,
			"state":              a.opts.DesiredState,
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
	opts    ProcessAction
}

// AttachActionWaitProcessWin action to wait for process status on windows systems
func AttachActionWaitProcessWin(deploy deploy.Deployment, service deploy.ServiceRequest, actionOpts ProcessAction) deploy.ServiceOperation {
	return &actionWaitProcessWin{
		service: service,
		deploy:  deploy,
		opts:    actionOpts,
	}
}

// holds the json  output from powershell's Get-Process
type processInfoWin struct {
	ID        int    `json:"Id"`
	HasExited bool   `json:"HasExited"`
	Name      string `json:"Name"`
}

// Run executes the command
func (a *actionWaitProcessWin) Run(ctx context.Context) (string, error) {
	exp := utils.GetExponentialBackOff(a.opts.MaxTimeout)

	mustBePresent := false
	if a.opts.DesiredState == "started" {
		mustBePresent = true
	}
	retryCount := 1

	processStatus := func() error {
		log.WithFields(log.Fields{
			"desiredState": a.opts.DesiredState,
			"occurrences":  a.opts.Occurrences,
			"process":      a.opts.Process,
		}).Trace("Checking process desired state on the container")

		// Get-Process | select Name,HasExited,Id | ConvertTo-Json
		cmds := []string{"powershell.exe", fmt.Sprintf("Get-Process %s | select Name,HasExited,Id | ConvertTo-Json", a.opts.Process)}
		output, err := a.deploy.ExecIn(ctx, deploy.NewServiceRequest(common.FleetProfileName), a.service, cmds)
		if err != nil {
			log.WithField("error", err).Error("unable to get process output")
			retryCount++
			return err
		}
		var processList []processInfoWin
		if err = json.Unmarshal([]byte(output), &processList); err != nil {
			log.WithField("error", err).Trace("Failed to unmarshal JSON output, will retry with single entry")
			var processEntry processInfoWin
			if err = json.Unmarshal([]byte(output), &processEntry); err != nil {
				log.WithField("error", err).Trace("Failed to unmarshal JSON output, will retry with single entry")
			}
			retryCount++
			return err
		}
		log.WithField("processList", processList).Trace("Process list")

		desiredStatePids := []int{}
		for _, processItem := range processList {
			log.WithFields(log.Fields{
				"desiredState":  a.opts.DesiredState,
				"mustBePresent": mustBePresent,
				"pid":           processItem.ID,
				"hasExited":     processItem.HasExited,
				"process":       a.opts.Process,
			}).Tracef("Checking if process is in the S state")

			if mustBePresent && strings.EqualFold(a.opts.Process, processItem.Name) && !processItem.HasExited {
				desiredStatePids = append(desiredStatePids, processItem.ID)
			} else if !mustBePresent && strings.EqualFold(a.opts.Process, processItem.Name) {
				desiredStatePids = append(desiredStatePids, processItem.ID)
			}
		}

		occurrencesMatched := (len(desiredStatePids) == a.opts.Occurrences)

		// both true or both false
		if mustBePresent == occurrencesMatched {
			log.WithFields(log.Fields{
				"desiredOccurrences": a.opts.Occurrences,
				"desiredState":       a.opts.DesiredState,
				"service":            a.service,
				"mustBePresent":      mustBePresent,
				"occurrences":        len(desiredStatePids),
				"process":            a.opts.Process,
			}).Infof("Process desired state checked")

			return nil
		}

		if mustBePresent {
			err = fmt.Errorf("%s process is not running in the OS with the desired number of occurrences (%d) yet", a.opts.Process, a.opts.Occurrences)
			log.WithFields(log.Fields{
				"desiredOccurrences": a.opts.Occurrences,
				"desiredState":       a.opts.DesiredState,
				"elapsedTime":        exp.GetElapsedTime(),
				"error":              err,
				"service":            a.service,
				"occurrences":        len(desiredStatePids),
				"process":            a.opts.Process,
				"retry":              retryCount,
			}).Warn(err.Error())

			retryCount++

			return err
		}

		err = fmt.Errorf("%s process is still running in the OS", a.opts.Process)
		log.WithFields(log.Fields{
			"desiredOccurrences": a.opts.Occurrences,
			"elapsedTime":        exp.GetElapsedTime(),
			"error":              err,
			"service":            a.service,
			"occurrences":        len(desiredStatePids),
			"process":            a.opts.Process,
			"state":              a.opts.DesiredState,
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
