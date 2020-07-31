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
	Init() error
	InstallChart(name string, chart string, version string, flags []string) error
}

// HelmFactory returns oone of the Helm supported versions, or an error
func HelmFactory(version string) (HelmManager, error) {
	if strings.HasPrefix(version, "3.") {
		helm := &helm3X{}
		helm.Version = version
		return helm, nil
	} else if strings.HasPrefix(version, "2.") {
		helm := &helm2X{}
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

func (h *helm3X) AddRepo(repo string, URL string) error {
	// use chart as application name
	args := []string{
		"repo", "add", repo, URL,
	}

	output, err := helmExecute(args...)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   repo,
		"url":    URL,
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

func (h *helm3X) Init() error {
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
	}).Debug("Chart installed")

	return nil
}

type helm2X struct {
	helm
}

func (h *helm2X) AddRepo(repo string, URL string) error {
	// use chart as application name
	args := []string{
		"repo", "add", repo, URL,
	}

	output, err := helmExecute(args...)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   repo,
		"url":    URL,
	}).Debug(repo + " Helm charts added")
	return nil
}

func (h *helm2X) DeleteChart(chart string) error {
	args := []string{
		"delete", "--purge", chart,
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

func (h *helm2X) Init() error {
	args := []string{"create", "-n", "kube-system", "serviceaccount", "tiller"}
	output, err := shell.Execute(".", "kubectl", args...)
	if err != nil {
		log.Error("Could not create Tiller ServiceAccount")
		return err
	}
	log.WithFields(log.Fields{
		"output": output,
	}).Debug("Tiller ServiceAccount created")

	args = []string{"create", "clusterrolebinding", "tiller", "--clusterrole=cluster-admin", "--serviceaccount=kube-system:tiller"}
	output, err = shell.Execute(".", "kubectl", args...)
	if err != nil {
		log.Error("Could not create Tiller ClusterRoleBinding")
		return err
	}
	log.WithFields(log.Fields{
		"output": output,
	}).Debug("Tiller ClusterRoleBinding created")

	args = []string{
		"init", "--wait", "--service-account", "tiller",
	}

	_, err = helmExecute(args...)
	if err != nil {
		return err
	}
	log.Debug("Helm initialised")
	return nil
}

func (h *helm2X) InstallChart(name string, chart string, version string, flags []string) error {
	log.WithFields(log.Fields{
		"chart":   chart,
		"name":    name,
		"version": version,
	}).Debug("Installing chart")

	args := []string{
		"install", chart, "--name", name, "--version", version,
	}
	for _, flag := range flags {
		if strings.HasPrefix(flag, "--timeout=") {
			log.WithFields(log.Fields{
				"flag": flag,
			}).Debug("Sanitising flag format for Helm 2")
			timeoutValue := strings.Split(flag, "=")
			args = append(args, timeoutValue[0])
			args = append(args, timeoutValue[1])
			continue
		}

		args = append(args, flag)
	}

	output, err := helmExecute(args...)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"chart":   chart,
		"name":    name,
		"output":  output,
		"version": version,
	}).Debug("Chart installed")

	return nil
}

func helmExecute(args ...string) (string, error) {
	output, err := shell.Execute(".", "helm", args...)
	if err != nil {
		return "", err
	}

	return output, nil
}
