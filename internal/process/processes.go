// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/elastic/e2e-testing/internal/action"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

// CheckState checks if a process is in the desired state in a container
// name of the container for the service:
// we are using the underlying deployer to run the commands in the container/service
func CheckState(ctx context.Context, deployer deploy.Deployment, service deploy.ServiceRequest, process string, state string, occurrences int) error {
	timeout := time.Duration(utils.TimeoutFactor) * time.Minute

	if runtime.GOOS == "windows" {
		process = fmt.Sprintf("%s.exe", process)
	}

	actionOpts := action.ProcessAction{
		Process:      process,
		DesiredState: state,
		Occurrences:  occurrences,
		MaxTimeout:   timeout}
	waitForProcess, err := action.Attach(ctx, deployer, service, action.ActionWaitForProcess, actionOpts)
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
