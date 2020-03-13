package services

import (
	"errors"
	"strings"

	"github.com/elastic/metricbeat-tests-poc/cli/shell"
	log "github.com/sirupsen/logrus"
)

// HelmManager defines the operations for Helm
type HelmManager interface {
	AddRepo(repo string, URL string) error
	DeleteChart(chart string) error
	Init() error
	InstallChart(name string, chart string, version string) error
}

// HelmFactory returns oone of the Helm supported versions, or an error
func HelmFactory(version string) (HelmManager, error) {
	if strings.HasPrefix(version, "3.1.") {
		helm := &helm3_1{}
		helm.Version = version
		return helm, nil
	} else if strings.HasPrefix(version, "2.16.") {
		helm := &helm2_16{}
		helm.Version = version
		return helm, nil
	}

	var helm HelmManager
	return helm, errors.New("Sorry, we don't support Helm v" + version + " version")
}

type helm struct {
	Version string
}

type helm3_1 struct {
	helm
}

func (h *helm3_1) AddRepo(repo string, URL string) error {
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

func (h *helm3_1) DeleteChart(chart string) error {
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

func (h *helm3_1) Init() error {
	return nil
}

func (h *helm3_1) InstallChart(name string, chart string, version string) error {
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

type helm2_16 struct {
	helm
}

func (h *helm2_16) AddRepo(repo string, URL string) error {
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

func (h *helm2_16) DeleteChart(chart string) error {
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

func (h *helm2_16) Init() error {
	args := []string{
		"init", "--namespace", "default",
	}

	_, err := helmExecute(args...)
	if err != nil {
		return err
	}
	log.Debug("Helm initialised")
	return nil
}

func (h *helm2_16) InstallChart(name string, chart string, version string) error {
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
