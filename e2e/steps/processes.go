// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package steps

import (
	"time"

	"github.com/elastic/e2e-testing/e2e"
	log "github.com/sirupsen/logrus"
)

// CheckProcessStateOnTheHost checks if a process is in the desired state in a container
// name of the container for the service:
// we are using the Docker client instead of docker-compose
// because it does not support returning the output of a
// command: it simply returns error level
func CheckProcessStateOnTheHost(containerName string, process string, state string, timeoutFactor int) error {
	timeout := time.Duration(timeoutFactor) * time.Minute

	err := e2e.WaitForProcess(containerName, process, state, timeout)
	if err != nil {
		if state == "started" {
			log.WithFields(log.Fields{
				"container ": containerName,
				"error":      err,
				"timeout":    timeout,
			}).Error("The process was not found but should be present")
		} else {
			log.WithFields(log.Fields{
				"container": containerName,
				"error":     err,
				"timeout":   timeout,
			}).Error("The process was found but shouldn't be present")
		}

		return err
	}

	return nil
}
