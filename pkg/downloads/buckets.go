// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package downloads

import (
	"fmt"
	"strings"
)

// BeatsCIArtifactsBase name of the bucket used to store the artifacts
const BeatsCIArtifactsBase = "beats-ci-artifacts"

// BucketURLResolver interface to resolve URL for artifacts in a bucket
type BucketURLResolver interface {
	Resolve() (string, string, string)
}

// BeatsLegacyURLResolver resolver for legacy Beats projects, such as metricbeat, filebeat, etc
// The Elastic Agent must use the project resolver
type BeatsLegacyURLResolver struct {
	Bucket   string
	Beat     string
	Variant  string
	FileName string
}

// NewBeatsLegacyURLResolver creates a new resolver for Beats projects
// The Elastic Agent must use the project resolver
func NewBeatsLegacyURLResolver(beat string, fileName string, variant string) *BeatsLegacyURLResolver {
	return &BeatsLegacyURLResolver{
		Bucket:   BeatsCIArtifactsBase,
		Beat:     beat,
		FileName: fileName,
		Variant:  variant,
	}
}

// Resolve returns the bucket, prefix and object for Beats artifacts
func (r *BeatsLegacyURLResolver) Resolve() (string, string, string) {
	artifact := r.Beat
	fileName := r.FileName

	if strings.EqualFold(r.Variant, "ubi8") {
		artifact = strings.ReplaceAll(artifact, "-ubi8", "")
	}

	prefix := fmt.Sprintf("snapshots/%s", artifact)
	object := fileName

	// the commit SHA will identify univocally the artifact in the GCP storage bucket
	if GithubCommitSha1 != "" {
		prefix = fmt.Sprintf("commits/%s", GithubCommitSha1)
		object = artifact + "/" + fileName
	}

	return r.Bucket, prefix, object
}

// BeatsURLResolver resolver for Beats projects, such as metricbeat, filebeat, etc
// The Elastic Agent must use the project resolver
type BeatsURLResolver struct {
	Bucket   string
	Beat     string
	Variant  string
	FileName string
}

// NewBeatsURLResolver creates a new resolver for Beats projects
// The Elastic Agent must use the project resolver
func NewBeatsURLResolver(beat string, fileName string, variant string) *BeatsURLResolver {
	return &BeatsURLResolver{
		Bucket:   BeatsCIArtifactsBase,
		Beat:     beat,
		FileName: fileName,
		Variant:  variant,
	}
}

// Resolve returns the bucket, prefix and object for Beats artifacts
func (r *BeatsURLResolver) Resolve() (string, string, string) {
	artifact := r.Beat
	fileName := r.FileName

	if strings.EqualFold(r.Variant, "ubi8") {
		artifact = strings.ReplaceAll(artifact, "-ubi8", "")
	}

	prefix := fmt.Sprintf("beats/snapshots/%s", artifact)
	object := fileName

	// the commit SHA will identify univocally the artifact in the GCP storage bucket
	if GithubCommitSha1 != "" {
		prefix = fmt.Sprintf("beats/commits/%s", GithubCommitSha1)
		object = artifact + "/" + fileName
	}

	return r.Bucket, prefix, object
}

// ProjectURLResolver resolver for Elastic projects, such as elastic-agent, fleet-server, etc.
// The Elastic Agent and Fleet Server must use the project resolver
type ProjectURLResolver struct {
	Bucket   string
	Project  string
	FileName string
}

// NewProjectURLResolver creates a new resolver for Elastic projects
// The Elastic Agent and Fleet Server must use the project resolver
func NewProjectURLResolver(project string, fileName string) *ProjectURLResolver {
	return &ProjectURLResolver{
		Bucket:   BeatsCIArtifactsBase,
		Project:  project,
		FileName: fileName,
	}
}

// Resolve returns the bucket, prefix and object for Elastic artifacts
func (r *ProjectURLResolver) Resolve() (string, string, string) {
	prefix := fmt.Sprintf("%s/snapshots", r.Project)

	// the commit SHA will identify univocally the artifact in the GCP storage bucket
	if GithubCommitSha1 != "" {
		prefix = fmt.Sprintf("%s/commits/%s", r.Project, GithubCommitSha1)
	}

	return r.Bucket, prefix, r.FileName
}
