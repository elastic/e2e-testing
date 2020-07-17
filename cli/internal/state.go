// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

// stateRun represents a Run
type stateRun struct {
	ID       string            // ID of the run
	Profile    stateService      // profile of the run (Optional)
	Env      map[string]string // environment for the run
	Services []stateService    // services in the run
}

// stateService represents a service in a Run
type stateService struct {
	Name string
}

// Recover recovers the state for a run
func Recover(id string, workdir string) map[string]string {
	run := stateRun{
		Env: map[string]string{},
	}

	stateFile := filepath.Join(workdir, id+".run")
	bytes, err := ReadFile(stateFile) //nolint
	if err != nil {
		return run.Env
	}

	err = yaml.Unmarshal(bytes, &run)
	if err != nil {
		log.WithFields(log.Fields{
			"stateFile": stateFile,
		}).Error("Could not unmarshal state")
	}

	return run.Env
}

// Destroy destroys the state for a run
func Destroy(id string, workdir string) {
	stateFile := filepath.Join(workdir, id+".run")
	err := os.Remove(stateFile)
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"stateFile": stateFile,
		}).Warn("Could not destroy state")

		return
	}

	log.WithFields(log.Fields{
		"stateFile": stateFile,
	}).Debug("State destroyed")
}

// Update updates the state of en execution, using ID as the file name for the run.
// The state file will be located under 'workdir', which by default will be the tool's
// workspace.
func Update(id string, workdir string, composeFilePaths []string, env map[string]string) {
	stateFile := filepath.Join(workdir, id+".run")

	log.WithFields(log.Fields{
		"dir":       workdir,
		"stateFile": stateFile,
	}).Debug("Updating state")

	run := stateRun{
		ID:       id,
		Env:      env,
		Services: []stateService{},
	}

	if strings.HasSuffix(id, "-profile") {
		run.Profile = stateService{
			Name: filepath.Base(filepath.Dir(composeFilePaths[0])),
		}
	}

	args := []string{}
	for i, f := range composeFilePaths {
		args = append(args, "-f", f)

		if i > 0 {
			run.Services = append(run.Services, stateService{
				Name: filepath.Base(filepath.Dir(f)),
			})
		}
	}
	args = append(args, "config")

	bytes, err := yaml.Marshal(&run)
	if err != nil {
		log.WithFields(log.Fields{
			"stateFile": stateFile,
		}).Error("Could not marshal state")
	}

	err = WriteFile(bytes, stateFile) //nolint
	if err != nil {
		log.WithFields(log.Fields{
			"stateFile": stateFile,
		}).Error("Could not create state file")
	}

	log.WithFields(log.Fields{
		"dir":       workdir,
		"stateFile": stateFile,
	}).Debug("State updated")
}
