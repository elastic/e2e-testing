// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package action

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/shirou/gopsutil/v3/process"
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
		processes, err := process.Processes()

		log.WithFields(log.Fields{
			"desiredState": a.opts.DesiredState,
			"occurrences":  a.opts.Occurrences,
			"process":      a.opts.Process,
		}).Trace("Checking process desired state on the container")

		desiredStatePids := []int32{}

		for _, p := range processes {
			processName, _ := p.Name()
			pid := p.Pid
			status, _ := p.Status()
			ppid, _ := p.Ppid()
			cmd, _ := p.Cmdline()
			isRunning, _ := p.IsRunning()

			if strings.EqualFold(processName, a.opts.Process) {
				log.WithFields(log.Fields{
					"name":      processName,
					"pid":       pid,
					"ppid":      ppid,
					"cmd":       cmd,
					"isRunning": isRunning,
					"status":    status,
				}).Trace("Checking Process")
				if mustBePresent && strings.EqualFold(status[0], "sleep") {
					desiredStatePids = append(desiredStatePids, pid)
				} else if !mustBePresent {
					desiredStatePids = append(desiredStatePids, pid)
				}
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
			}).Infof("Process desired state found")
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
