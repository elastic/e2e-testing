// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package downloads

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/curl"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

// DownloadURLResolver interface to resolve URLs for downloadable artifacts
type DownloadURLResolver interface {
	Resolve() (url string, shaURL string, err error)
}

// ArtifactURLResolver type to resolve the URL of artifacts that are currently in development, from the artifacts API
type ArtifactURLResolver struct {
	FullName string
	Name     string
	Version  string
}

// NewArtifactURLResolver creates a new resolver for artifacts that are currently in development, from the artifacts API
func NewArtifactURLResolver(fullName string, name string, version string) DownloadURLResolver {
	// resolve version alias
	resolvedVersion, err := GetElasticArtifactVersion(version)
	if err != nil {
		return nil
	}

	fullName = strings.ReplaceAll(fullName, version, resolvedVersion)

	return &ArtifactURLResolver{
		FullName: fullName,
		Name:     name,
		Version:  resolvedVersion,
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
	if downloadObject == nil {
		log.WithFields(log.Fields{
			"artifact": artifact,
			"name":     artifactName,
			"version":  version,
		}).Error("object not found in Artifact API")
		return "", "", fmt.Errorf("object not found in Artifact API")
	}

	downloadURL, ok := downloadObject.Path("url").Data().(string)
	if !ok {
		return "", "", fmt.Errorf("key 'url' does not exist for artifact %s", artifact)
	}
	downloadshaURL, ok := downloadObject.Path("sha_url").Data().(string)
	if !ok {
		return "", "", fmt.Errorf("key 'sha_url' does not exist for artifact %s", artifact)
	}

	return downloadURL, downloadshaURL, nil
}

type ArtifactsSnapshotVersion struct {
	Host string
}

func newArtifactsSnapshotCustom(host string) *ArtifactsSnapshotVersion {
	return &ArtifactsSnapshotVersion{
		Host: host,
	}
}

// Uses artifacts-snapshot.elastic.co to retrieve the latest version of a SNAPSHOT artifact
func NewArtifactsSnapshot() *ArtifactsSnapshotVersion {
	return &ArtifactsSnapshotVersion{
		Host: "https://artifacts-snapshot.elastic.co",
	}
}

// GetSnapshotArtifactVersion returns the current version:
// Uses artifacts-snapshot.elastic.co to retrieve the latest version of a SNAPSHOT artifact
// 1. Elastic's artifact repository, building the JSON path query based
// If the version is a SNAPSHOT including a commit, then it will directly use the version without checking the artifacts API
// i.e. GetSnapshotArtifactVersion("$VERSION-abcdef-SNAPSHOT")
func (as *ArtifactsSnapshotVersion) GetSnapshotArtifactVersion(version string) (string, error) {
	cacheKey := fmt.Sprintf("%s/beats/latest/%s.json", as.Host, version)

	if val, ok := elasticVersionsCache[cacheKey]; ok {
		log.WithFields(log.Fields{
			"URL":     cacheKey,
			"version": val,
		}).Debug("Retrieving version from local cache")
		return val, nil
	}

	if SnapshotHasCommit(version) {
		elasticVersionsCache[cacheKey] = version
		return version, nil
	}

	exp := utils.GetExponentialBackOff(time.Minute)

	retryCount := 1

	body := ""

	apiStatus := func() error {
		r := curl.HTTPRequest{
			URL: cacheKey,
		}

		response, err := curl.Get(r)
		if err != nil {
			log.WithFields(log.Fields{
				"version":        version,
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
		return "", err
	}

	type ArtifactsSnapshotResponse struct {
		Version     string `json:"version"`      // example value: "8.8.3-SNAPSHOT"
		BuildID     string `json:"build_id"`     // example value: "8.8.3-b1d8691a"
		ManifestURL string `json:"manifest_url"` // example value: https://artifacts-snapshot.elastic.co/beats/8.8.3-b1d8691a/manifest-8.8.3-SNAPSHOT.json
		SummaryURL  string `json:"summary_url"`  // example value: https://artifacts-snapshot.elastic.co/beats/8.8.3-b1d8691a/summary-8.8.3-SNAPSHOT.html
	}
	response := ArtifactsSnapshotResponse{}
	err = json.Unmarshal([]byte(body), &response)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": version,
			"body":    body,
		}).Error("Could not parse the response body to retrieve the version")

		return "", fmt.Errorf("could not parse the response body to retrieve the version: %w", err)
	}

	hashParts := strings.Split(response.BuildID, "-")
	if (len(hashParts) < 2) || (hashParts[1] == "") {
		log.WithFields(log.Fields{
			"buildId": response.BuildID,
		}).Error("Could not parse the build_id to retrieve the version hash")
		return "", fmt.Errorf("could not parse the build_id to retrieve the version hash: %s", response.BuildID)
	}
	hash := hashParts[1]
	parsedVersion := hashParts[0]

	latestVersion := fmt.Sprintf("%s-%s-SNAPSHOT", parsedVersion, hash)

	log.WithFields(log.Fields{
		"alias":   version,
		"version": latestVersion,
	}).Debug("Latest version for current version obtained")

	elasticVersionsCache[cacheKey] = latestVersion

	return latestVersion, nil
}

// NewArtifactURLResolver creates a new resolver for artifacts that are currently in development, from the artifacts API
func NewArtifactSnapshotURLResolver(fullName string, name string, version string) DownloadURLResolver {
	// resolve version alias
	resolvedVersion, err := NewArtifactsSnapshot().GetSnapshotArtifactVersion(version)
	if err != nil {
		return nil
	}

	// fullName = strings.ReplaceAll(fullName, version, resolvedVersion)

	return &ArtifactsSnapshotURLResolver{
		FullName:        fullName,
		Name:            name,
		Version:         resolvedVersion,
		SnapshotApiHost: "https://artifacts-snapshot.elastic.co",
	}
}

// ArtifactsSnapshotURLResolver type to resolve the URL of artifacts that are currently in development, from the artifacts API
type ArtifactsSnapshotURLResolver struct {
	FullName        string
	Name            string
	Version         string
	SnapshotApiHost string
}

func (asur *ArtifactsSnapshotURLResolver) Resolve() (string, string, error) {
	artifactName := asur.FullName
	artifact := asur.Name
	version := asur.Version
	commit, err := ExtractCommitHash(version)
	semVer := GetVersion(version)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":     artifact,
			"artifactName": artifactName,
			"version":      version,
		}).Info("The version does not contain a commit hash, it is not a snapshot")
		return "", "", err
	}

	exp := utils.GetExponentialBackOff(time.Minute)

	retryCount := 1

	body := ""

	apiStatus := func() error {
		r := curl.HTTPRequest{
			// https://artifacts-snapshot.elastic.co/beats/8.9.0-d1b14479/manifest-8.9.0-SNAPSHOT.json
			URL: fmt.Sprintf("https://artifacts-snapshot.elastic.co/beats/%s-%s/manifest-%s-SNAPSHOT.json", semVer, commit, semVer),
		}

		response, err := curl.Get(r)
		if err != nil {
			log.WithFields(log.Fields{
				"artifact":       artifact,
				"artifactName":   artifactName,
				"version":        version,
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

	err = backoff.Retry(apiStatus, exp)
	if err != nil {
		return "", "", err
	}

	var jsonParsed map[string]interface{}
	err = json.Unmarshal([]byte(body), &jsonParsed)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":     artifact,
			"artifactName": artifactName,
			"version":      version,
		}).Error("Could not parse the response body for the artifact")
		return "", "", err
	}

	url, shaURL, err := findSnapshotPackage(jsonParsed, artifactName)
	if err != nil {
		return "", "", err
	}

	log.WithFields(log.Fields{
		"retries":      retryCount,
		"artifact":     artifact,
		"artifactName": artifactName,
		"elapsedTime":  exp.GetElapsedTime(),
		"version":      version,
	}).Trace("Artifact found")

	return url, shaURL, nil
}

func findSnapshotPackage(jsonParsed map[string]interface{}, fullName string) (string, string, error) {
	projects, ok := jsonParsed["projects"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("key 'projects' does not exist")
	}

	for _, project := range projects {
		projectPackages, ok := project.(map[string]interface{})["packages"].(map[string]interface{})
		if !ok {
			continue
		}

		pack, ok := projectPackages[fullName].(map[string]interface{})

		if !ok {
			continue
		}

		return pack["url"].(string), pack["sha_url"].(string), nil

	}
	return "", "", fmt.Errorf("package %s not found", fullName)
}

// ReleaseURLResolver type to resolve the URL of downloads that are currently published in elastic.co/downloads
type ReleaseURLResolver struct {
	Project  string
	FullName string
	Name     string
}

// NewReleaseURLResolver creates a new resolver for downloads that are currently published in elastic.co/downloads
func NewReleaseURLResolver(project string, fullName string, name string) *ReleaseURLResolver {
	return &ReleaseURLResolver{
		FullName: fullName,
		Name:     name,
		Project:  project,
	}
}

// Resolve resolves the URL of a download, which is located in the Elastic. It will use a HEAD request
// and if it returns a 200 OK it will return the URL of both file and its SHA512 file
func (r *ReleaseURLResolver) Resolve() (string, string, error) {
	url := fmt.Sprintf("https://artifacts.elastic.co/downloads/%s/%s/%s", r.Project, r.Name, r.FullName)
	shaURL := fmt.Sprintf("%s.sha512", url)

	exp := utils.GetExponentialBackOff(time.Minute)
	retryCount := 1
	found := false

	apiStatus := func() error {
		r := curl.HTTPRequest{
			URL: url,
		}

		_, err := curl.Head(r)
		if err != nil {
			if strings.EqualFold(err.Error(), "HEAD request failed with 404") {
				log.WithFields(log.Fields{
					"resolver":       r,
					"error":          err,
					"retry":          retryCount,
					"statusEndpoint": r.URL,
					"elapsedTime":    exp.GetElapsedTime(),
				}).Debug("Download could not be found at the Elastic downloads API")
				return nil
			}

			log.WithFields(log.Fields{
				"resolver":       r,
				"error":          err,
				"retry":          retryCount,
				"statusEndpoint": r.URL,
				"elapsedTime":    exp.GetElapsedTime(),
			}).Debug("The Elastic downloads API is not available yet")

			retryCount++

			return err
		}

		found = true
		log.WithFields(log.Fields{
			"retries":        retryCount,
			"statusEndpoint": r.URL,
			"elapsedTime":    exp.GetElapsedTime(),
		}).Info("Download was found in the Elastic downloads API")

		return nil
	}

	err := backoff.Retry(apiStatus, exp)
	if err != nil {
		return "", "", err
	}

	if !found {
		return "", "", fmt.Errorf("download could not be found at the Elastic downloads API")
	}

	return url, shaURL, nil
}
