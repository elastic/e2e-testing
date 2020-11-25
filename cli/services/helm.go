// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package services

import (
	"errors"
	"strings"

	"github.com/elastic/e2e-testing/cli/shell"
	log "github.com/sirupsen/logrus"
)

// HelmManager defines the operations for Helm
type HelmManager interface {
	AddRepo(repo string, URL string) error
	DeleteChart(chart string) error
	InstallChart(name string, chart string, version string, flags []string) error
}

// HelmFactory returns oone of the Helm supported versions, or an error
func HelmFactory(version string) (HelmManager, error) {
	if strings.HasPrefix(version, "3.") {
		helm := &helm3X{}
		helm.Version = version
		return helm, nil
	}

	var helm HelmManager
	return helm, errors.New("Sorry, we don't support Helm v" + version + " version")
}

type helm struct {
	Version string
}

type helm3X struct {
	helm
}

func (h *helm3X) AddRepo(repo string, url string) error {
	// use chart as application name
	args := []string{
		"repo", "add", repo, url,
	}

	output, err := helmExecute(args...)
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

func (h *helm3X) DeleteChart(chart string) error {
	args := []string{
		"delete", chart,
	}

	output, err := helmExecute(args...)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"output": output,
		"chart":  chart,
	}).Debug("Chart deleted")
	return nil
}

func (h *helm3X) InstallChart(name string, chart string, version string, flags []string) error {
	args := []string{
		"install", name, chart, "--version", version,
	}
	args = append(args, flags...)

	output, err := helmExecute(args...)
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

func helmExecute(args ...string) (string, error) {
	output, err := shell.Execute(".", "helm", args...)
	if err != nil {
		return "", err
	}

	return output, nil
}
