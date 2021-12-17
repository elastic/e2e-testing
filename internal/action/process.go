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
	deploy  deploy.Deployer
	opts    ProcessAction
}

// AttachActionWaitProcess action to woit for a process status on *nix like systems
func AttachActionWaitProcess(deploy deploy.Deployer, service deploy.ServiceRequest, actionOpts ProcessAction) deploy.ServiceOperation {
	return &actionWaitProcess{
		service: service,
		deploy:  deploy,
		opts:    actionOpts,
	}
}

// Run executes the command
func (a *actionWaitProcess) Run(ctx context.Context) (string, error) {
	exp := utils.GetExponentialBackOff(a.opts.MaxTimeout)

	pidState := "stopped"
	if a.opts.DesiredState == "started" {
		pidState = "sleep"
	}
	retryCount := 1

	processStatus := func() error {
		processes, err := process.Processes()

		desiredStatePids := []int32{}

		for _, p := range processes {
			processName, _ := p.Name()
			pid := p.Pid
			status, _ := p.Status()
			ppid, _ := p.Ppid()
			cmd, _ := p.Cmdline()

			if strings.EqualFold(processName, a.opts.Process) && strings.EqualFold(status[0], pidState) {
				desiredStatePids = append(desiredStatePids, pid)
				log.WithFields(log.Fields{
					"name":               processName,
					"pid":                pid,
					"ppid":               ppid,
					"cmd":                cmd,
					"status":             status,
					"desiredState":       a.opts.DesiredState,
					"desiredOccurrences": a.opts.Occurrences,
					"foundOccurrences":   len(desiredStatePids),
				}).Debug("Checking Process desired state")
			}
		}

		occurrencesMatched := (len(desiredStatePids) == a.opts.Occurrences)

		// both true or both false
		if occurrencesMatched {
			log.WithFields(log.Fields{
				"desiredOccurrences": a.opts.Occurrences,
				"foundOccurrences":   len(desiredStatePids),
				"desiredState":       a.opts.DesiredState,
				"service":            a.service,
				"process":            a.opts.Process,
			}).Infof("Process desired state found")
			return nil
		}
		err = fmt.Errorf("%s process is not in the desiredState the desired number of occurrences (%d) yet", a.opts.Process, a.opts.Occurrences)
		log.WithFields(log.Fields{
			"desiredOccurrences": a.opts.Occurrences,
			"foundOccurrences":   len(desiredStatePids),
			"desiredState":       a.opts.DesiredState,
			"elapsedTime":        exp.GetElapsedTime(),
			"service":            a.service,
			"process":            a.opts.Process,
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
