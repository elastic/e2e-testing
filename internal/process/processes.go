// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/shirou/gopsutil/v3/process"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

const (
	// actionWaitForProcess const for choosing the wait for process action
	actionWaitForProcess = "wait-for-process"
)

// actionOpt contains the necessary options to pass into process action
type actionOpt struct {
	Process      string
	DesiredState string
	Occurrences  int
	MaxTimeout   time.Duration
}

// actionWait implements operation for waiting on a process status
type actionWait struct {
	service deploy.ServiceRequest
	deploy  deploy.Deployment
	opts    actionOpt
}

// attachActionWait action to woit for a process status on *nix like systems
func attachActionWait(deploy deploy.Deployment, service deploy.ServiceRequest, actionOpts actionOpt) deploy.ServiceOperation {
	return &actionWait{
		service: service,
		deploy:  deploy,
		opts:    actionOpts,
	}
}

// attach will attach a service operator action to service operator
func attach(ctx context.Context, deploy deploy.Deployment, service deploy.ServiceRequest, action string, actionOpts interface{}) (deploy.ServiceOperation, error) {
	span, _ := apm.StartSpanOptions(ctx, "Attaching action to service operator", "action.attach", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	log.WithFields(log.Fields{
		"service": service,
		"action":  action,
	}).Trace("Attaching action for service")

	switch action {
	case actionWaitForProcess:
		newActionOpts, ok := actionOpts.(actionOpt)
		if !ok {
			log.Fatal("Unable to cast to action options to actionOpt")
		}
		attachAction := attachActionWait(deploy, service, newActionOpts)
		return attachAction, nil
	}

	log.WithField("action", action).Warn("Unknown action called")
	return nil, nil
}

// CheckState checks if a process is in the desired state in a container
// name of the container for the service:
// we are using the underlying deployer to run the commands in the container/service
func CheckState(ctx context.Context, deployer deploy.Deployment, service deploy.ServiceRequest, process string, state string, occurrences int) error {
	timeout := time.Duration(utils.TimeoutFactor) * time.Minute

	if runtime.GOOS == "windows" {
		process = fmt.Sprintf("%s.exe", process)
	}

	actionOpts := actionOpt{
		Process:      process,
		DesiredState: state,
		Occurrences:  occurrences,
		MaxTimeout:   timeout}
	waitForProcess, err := attach(ctx, deployer, service, actionWaitForProcess, actionOpts)
	if err != nil {
		log.WithField("error", err).Error("Unable to attach Process check action")
	}

	_, err = waitForProcess.Run(ctx)
	if err != nil {
		if state == "started" {
			log.WithFields(log.Fields{
				"service ": service,
				"error":    err,
				"process ": process,
				"timeout":  timeout,
			}).Error("The process was not found but should be present")
		} else {
			log.WithFields(log.Fields{
				"service":  service,
				"error":    err,
				"process ": process,
				"timeout":  timeout,
			}).Error("The process was found but shouldn't be present")
		}

		return err
	}

	return nil
}

// Run executes the command
func (a *actionWait) Run(ctx context.Context) (string, error) {
	if a.service.IsContainer {
		// when we run the tests in a container, we need to execute the command inside the container
		return runInContainer(ctx, a)
	}

	exp := utils.GetExponentialBackOff(a.opts.MaxTimeout)

	pidState := "stopped"
	if a.opts.DesiredState == "started" {
		pidState = "sleep"
	}
	retryCount := 1

	processStatus := func() error {
		processes, err := process.Processes()
		if err != nil {
			return err
		}

		desiredStatePids := []int32{}

		for _, p := range processes {
			processName, _ := p.Name()
			pid := p.Pid
			status, _ := p.Status()
			ppid, _ := p.Ppid()
			cmd, _ := p.Cmdline()

			checkFunction := func() bool {
				if runtime.GOOS == "windows" {
					// Windows status is not supported at this moment by the library
					// - https://github.com/shirou/gopsutil#process-class
					// - https://github.com/shirou/gopsutil/issues/1016
					// for that reason we are only checking that the process is running
					isRunning, _ := p.IsRunning()

					if a.opts.DesiredState == "started" {
						return isRunning
					}
					return !isRunning
				}

				return strings.EqualFold(status[0], pidState)
			}

			if strings.EqualFold(processName, a.opts.Process) && checkFunction() {
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

// runInContainer restored from https://github.com/elastic/e2e-testing/pull/1740, it executes
// pgrep in the target container defined by the service of the actionWait
func runInContainer(ctx context.Context, a *actionWait) (string, error) {
	log.WithFields(log.Fields{
		"desiredState": a.opts.DesiredState,
		"occurrences":  a.opts.Occurrences,
		"process":      a.opts.Process,
	}).Trace("Checking for container")

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
