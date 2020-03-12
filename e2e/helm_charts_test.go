package e2e

import (
	"errors"
	"strings"

	shell "github.com/elastic/metricbeat-tests-poc/cli/shell"

	"github.com/cucumber/godog"
	log "github.com/sirupsen/logrus"
)

func init() {
	binaries := []string{
		"kubectl",
		"helm",
	}

	shell.CheckInstalledSoftware(binaries)
}

// HelmChartTestSuite represents a test suite for a helm chart
//nolint:unused
type HelmChartTestSuite struct {
	Name    string // the name of the chart
	Version string // the helm chart version for the test
}

// getFullName returns the name plus version, in lowercase, enclosed in quotes
func (ts *HelmChartTestSuite) getFullName() string {
	return strings.ToLower("'" + ts.Name + "-" + ts.Version + "'")
}

func (ts *HelmChartTestSuite) delete() error {
	args := []string{
		"delete", ts.Name,
	}

	output, err := shell.Execute(".", "helm", args...)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("Chart deleted")

	return nil
}

func (ts *HelmChartTestSuite) addElasticRepo() error {
	elasticHelmChartsURL := "https://helm.elastic.co"

	// use chart as application name
	args := []string{
		"repo", "add", "elastic", elasticHelmChartsURL,
	}

	output, err := shell.Execute(".", "helm", args...)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"output": output,
		"name":   "elastic",
		"url":    elasticHelmChartsURL,
	}).Debug("Elastic Helm charts added")

	return nil
}

func (ts *HelmChartTestSuite) install(chart string) error {
	ts.Name = chart

	elasticChart := "elastic/" + ts.Name

	// use chart as application name
	args := []string{
		"install", ts.Name, elasticChart,
	}

	output, err := shell.Execute(".", "helm", args...)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
		"chart":  elasticChart,
	}).Debug("Chart installed")

	return nil
}

func (ts *HelmChartTestSuite) aClusterIsRunning() error {
	return nil
}

func (ts *HelmChartTestSuite) elasticsHelmChartIsInstalled(chart string) error {
	return ts.install(chart)
}

func (ts *HelmChartTestSuite) podsManagedByDaemonSet() error {
	args := []string{
		"get", "daemonsets", "--namespace=default", "-l", "app=" + ts.Name + "-" + ts.Name, "-o", "jsonpath='{.items[0].metadata.labels.chart}'",
	}

	output, err := shell.Execute(".", "kubectl", args...)
	if err != nil {
		return err
	}
	if output != ts.getFullName() {
		return errors.New("There is no DaemonSet for the chart. Expected:" + ts.getFullName() + ", Actual: " + output)
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("Pods are managed by a DaemonSet")

	return nil
}

func (ts *HelmChartTestSuite) resourceWillManageAdditionalPodsForMetricsets(pod string) error {
	args := []string{
		"get", "deployments", ts.Name + "-" + ts.Name + "-metrics", "-o", "jsonpath='{.metadata.labels.chart}'",
	}

	output, err := shell.Execute(".", "kubectl", args...)
	if err != nil {
		return err
	}
	if output != ts.getFullName() {
		return errors.New("There is no Deployment for the chart. Expected:" + ts.getFullName() + ", Actual: " + output)
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("Pods are managed by a DaemonSet")

	return nil
}

func (ts *HelmChartTestSuite) willRetrieveSpecificMetrics(chartName string) error {
	return godog.ErrPending
}

func (ts *HelmChartTestSuite) aResourceContainsTheContent(resource string, content string) error {
	return godog.ErrPending
}

func (ts *HelmChartTestSuite) aResourceManagesRBAC(resource string) error {
	return godog.ErrPending
}

func HelmChartFeatureContext(s *godog.Suite) {
	testSuite := HelmChartTestSuite{
		Version: "7.6.1",
	}

	s.Step(`^a cluster is running$`, testSuite.aClusterIsRunning)
	s.Step(`^the "([^"]*)" Elastic\'s helm chart is installed$`, testSuite.elasticsHelmChartIsInstalled)
	s.Step(`^a pod will be deployed on each node of the cluster by a DaemonSet$`, testSuite.podsManagedByDaemonSet)
	s.Step(`^a "([^"]*)" will manage additional pods for metricsets querying internal services$`, testSuite.resourceWillManageAdditionalPodsForMetricsets)
	s.Step(`^a "([^"]*)" chart will retrieve specific Kubernetes metrics$`, testSuite.willRetrieveSpecificMetrics)
	s.Step(`^a "([^"]*)" resource contains the "([^"]*)" content$`, testSuite.aResourceContainsTheContent)
	s.Step(`^a "([^"]*)" resource manages RBAC$`, testSuite.aResourceManagesRBAC)

	s.BeforeSuite(func() {
		log.Debug("Before Suite...")
		testSuite.addElasticRepo()
	})
	s.BeforeScenario(func(interface{}) {
		log.Info("Before Helm scenario...")
	})
	s.AfterScenario(func(interface{}, error) {
		log.Debug("After Helm scenario...")
		testSuite.delete()
	})
}
