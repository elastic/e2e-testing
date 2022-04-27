// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package downloads

import (
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/curl"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

// ReleaseURLResolver interface to resolve URLs for release artifacts
type ReleaseURLResolver interface {
	Resolve() (url string, shaURL string, err error)
}

// ArtifactURLResolver type to resolve the URL of artifacts that are currently in development, from the artifacts API
type ArtifactURLResolver struct {
	FullName string
	Name     string
	Version  string
}

// NewArtifactURLResolver creates a new resolver for artifacts that are currently in development, from the artifacts API
func NewArtifactURLResolver(fullName string, name string, version string) *ArtifactURLResolver {
	return &ArtifactURLResolver{
		FullName: fullName,
		Name:     name,
		Version:  version,
	}
}

// Resolve returns the URL of a released artifact, which its full name is defined in the first argument,
// from Elastic's artifact repository, building the JSON path query based on the full name
func (r *ArtifactURLResolver) Resolve() (string, string, error) {
	artifactName := r.FullName
	artifact := r.Name
	version := r.Version

	exp := utils.GetExponentialBackOff(time.Minute)

	retryCount := 1

	body := ""

	tmpVersion := version
	hasCommit := SnapshotHasCommit(version)
	if hasCommit {
		log.WithFields(log.Fields{
			"version": version,
		}).Trace("Removing SNAPSHOT from version including commit")

		// remove the SNAPSHOT from the VERSION as the artifacts API supports commits in the version, but without the snapshot suffix
		tmpVersion = GetCommitVersion(version)
	}

	apiStatus := func() error {
		r := curl.HTTPRequest{
			URL: fmt.Sprintf("https://artifacts-api.elastic.co/v1/search/%s/%s?x-elastic-no-kpi=true", tmpVersion, artifact),
		}

		response, err := curl.Get(r)
		if err != nil {
			log.WithFields(log.Fields{
				"artifact":       artifact,
				"artifactName":   artifactName,
				"version":        tmpVersion,
				"error":          err,
				"retry":          retryCount,
				"statusEndpoint": r.URL,
				"elapsedTime":    exp.GetElapsedTime(),
			}).Warn("The Elastic artifacts API is not available yet")

			retryCount++

			return err
		}

		log.WithFields(log.Fields{
			"retries":        retryCount,
			"statusEndpoint": r.URL,
			"elapsedTime":    exp.GetElapsedTime(),
		}).Debug("The Elastic artifacts API is available")

		body = response
		return nil
	}

	err := backoff.Retry(apiStatus, exp)
	if err != nil {
		return "", "", err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":     artifact,
			"artifactName": artifactName,
			"version":      tmpVersion,
		}).Error("Could not parse the response body for the artifact")
		return "", "", err
	}

	log.WithFields(log.Fields{
		"retries":      retryCount,
		"artifact":     artifact,
		"artifactName": artifactName,
		"elapsedTime":  exp.GetElapsedTime(),
		"version":      tmpVersion,
	}).Trace("Artifact found")

	if hasCommit {
		// remove commit from the artifact as it comes like this: elastic-agent-8.0.0-abcdef-SNAPSHOT-darwin-x86_64.tar.gz
		artifactName = RemoveCommitFromSnapshot(artifactName)
	}

	packagesObject := jsonParsed.Path("packages")
	// we need to get keys with dots using Search instead of Path
	downloadObject := packagesObject.Search(artifactName)
	downloadURL := downloadObject.Path("url").Data().(string)
	downloadshaURL := downloadObject.Path("sha_url").Data().(string)

	return downloadURL, downloadshaURL, nil
}
