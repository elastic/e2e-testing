// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package helm

import (
	"context"
	"errors"
	"strings"

	"github.com/elastic/e2e-testing/internal/shell"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm/v2"
)

// Manager defines the operations for Helm
type Manager interface {
	AddRepo(ctx context.Context, repo string, URL string) error
	DeleteChart(ctx context.Context, chart string) error
	InstallChart(ctx context.Context, name string, chart string, version string, flags []string) error
}

// Factory returns oone of the Helm supported versions, or an error
func Factory(version string) (Manager, error) {
	if strings.HasPrefix(version, "3.") {
		helm := &helm3X{}
		helm.Version = version
		return helm, nil
	}

	var helm Manager
	return helm, errors.New("Sorry, we don't support Helm v" + version + " version")
}

type helm struct {
	Version string
}

type helm3X struct {
	helm
}

func (h *helm3X) AddRepo(ctx context.Context, repo string, url string) error {
	span, _ := apm.StartSpanOptions(ctx, "Adding Helm repository", "helm.repo.add", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	// use chart as application name
	args := []string{
		"repo", "add", repo, url,
	}

	output, err := helmExecute(ctx, args...)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   repo,
		"url":    url,
	}).Debug(repo + " Helm charts added")
	return nil
}

func (h *helm3X) DeleteChart(ctx context.Context, chart string) error {
	span, _ := apm.StartSpanOptions(ctx, "Deleting Helm chart", "helm.chart.delete", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	args := []string{
		"delete", chart,
	}

	output, err := helmExecute(ctx, args...)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"output": output,
		"chart":  chart,
	}).Debug("Chart deleted")
	return nil
}

func (h *helm3X) InstallChart(ctx context.Context, name string, chart string, version string, flags []string) error {
	span, _ := apm.StartSpanOptions(ctx, "Installing Helm chart", "helm.chart.install", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	args := []string{
		"install", name, chart, "--version", version,
	}
	args = append(args, flags...)

	output, err := helmExecute(ctx, args...)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"chart":   chart,
		"name":    name,
		"output":  output,
		"version": version,
	}).Info("Chart installed")

	return nil
}

func helmExecute(ctx context.Context, args ...string) (string, error) {
	output, err := shell.Execute(ctx, ".", "helm", args...)
	if err != nil {
		return "", err
	}

	return output, nil
}
