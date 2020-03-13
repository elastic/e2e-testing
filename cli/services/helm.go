package services

import (
	"errors"

	"github.com/elastic/metricbeat-tests-poc/cli/shell"
	log "github.com/sirupsen/logrus"
)

// HelmManager defines the operations for Helm
type HelmManager interface {
	AddRepo(repo string, URL string) error
	DeleteChart(chart string) error
	InstallChart(name string, chart string, version string) error
}

// HelmFactory returns oone of the Helm supported versions, or an error
func HelmFactory(version string) (HelmManager, error) {
	var helm HelmManager
	if version == "3.1.1" {
		return &helm3_1_1{}, nil
	}

	return helm, errors.New("Sorry, we don't support Helm v" + version + " version")
}

type helm3_1_1 struct {
}

func (h *helm3_1_1) AddRepo(repo string, URL string) error {
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

func (h *helm3_1_1) DeleteChart(chart string) error {
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

func (h *helm3_1_1) InstallChart(name string, chart string, version string) error {
	args := []string{
		"install", name, chart, "--version", version,
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
