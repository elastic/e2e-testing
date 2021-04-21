// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package steps

import (
	"path"

	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

// FetchBeatConfiguration where 'configFileName' represents the configuration file, including extension.
// It will read environment to determine if it must read a file from the local file system, or in the
// contrary, download a file from Github. In the latter case, it will use a commit of the maintenance branch
// used in the tests
func FetchBeatConfiguration(xpack bool, beat string, configFileName string) (string, error) {
	beatsLocalPath := shell.GetEnv("BEATS_LOCAL_PATH", "")

	if beatsLocalPath != "" {
		configurationFilePath := beatsLocalPath
		if xpack {
			configurationFilePath = path.Join(beatsLocalPath, "x-pack")
		}
		configurationFilePath = path.Join(beatsLocalPath, beat, configFileName)

		log.WithFields(log.Fields{
			"beat":  beat,
			"file":  configurationFilePath,
			"xpack": xpack,
		}).Trace("Reading configuration file from local path")

		return configurationFilePath, nil
	}

	refspec := shell.GetEnv("GITHUB_CHECK_SHA1", "master")

	configurationFileURL := "https://raw.githubusercontent.com/elastic/beats/" + refspec
	if xpack {
		configurationFileURL += "/x-pack"
	}

	configurationFileURL += "/" + beat + "/" + configFileName

	configurationFilePath, err := utils.DownloadFile(configurationFileURL)
	if err != nil {
		return "", err
	}

	log.WithFields(log.Fields{
		"beat":  beat,
		"path":  configurationFilePath,
		"url":   configurationFileURL,
		"xpack": xpack,
	}).Trace("Configuration file downloaded from Github")

	return configurationFilePath, nil
}
