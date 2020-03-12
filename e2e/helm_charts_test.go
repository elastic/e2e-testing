package e2e

import (
	"errors"
	"runtime"
	"strings"

	shell "github.com/elastic/metricbeat-tests-poc/cli/shell"

	"github.com/cucumber/godog"
	log "github.com/sirupsen/logrus"
)

func init() {
	binaries := []string{
		"kubectl",
		"helm",
		"minikube",
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

// getKubeStateName returns the kube-state-metrics name, in lowercase, enclosed in quotes
func (ts *HelmChartTestSuite) getKubeStateMetricsName() string {
	return strings.ToLower("'" + ts.Name + "-kube-state-metrics'")
}

// getKubeStateName returns the kube-state-metrics name, in lowercase, enclosed in quotes
func (ts *HelmChartTestSuite) getResourceName(resource string) string {
	if resource == "ClusterRole" {
		return strings.ToLower(ts.Name + "-" + ts.Name + "-cluster-role")
	} else if resource == "ClusterRoleBinding" {
		return strings.ToLower(ts.Name + "-" + ts.Name + "-cluster-role-binding")
	} else if resource == "ConfigMap" {
		return strings.ToLower(ts.Name + "-" + ts.Name + "-config")
	} else if resource == "ServiceAccount" {
		return strings.ToLower(ts.Name + "-" + ts.Name)
	}

	return ""
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

func (ts *HelmChartTestSuite) createCluster() error {
	flags := ""
	if runtime.GOOS == "linux" {
		// Minikube also supports a --vm-driver=none option that runs the Kubernetes components
		// on the host and not in a VM. Using this driver requires Docker and a Linux environment
		// but not a hypervisor.
		flags = "--vm-driver=none"
	}

	args := []string{"start", flags}

	output, err := shell.Execute(".", "minikube", args...)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("Cluster created")

	return nil
}

func (ts *HelmChartTestSuite) destroyCluster() error {
	args := []string{"delete"}

	output, err := shell.Execute(".", "minikube", args...)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("Cluster destroyed")

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
	return ts.createCluster()
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
		return errors.New("There is no DaemonSet for the " + ts.Name + " chart. Expected:" + ts.getFullName() + ", Actual: " + output)
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("A pod will be deployed on each node of the cluster by a DaemonSet")

	return nil
}

func (ts *HelmChartTestSuite) resourceWillManageAdditionalPodsForMetricsets(resource string) error {
	lowerResource := strings.ToLower(resource)
	args := []string{
		"get", lowerResource + "s", ts.Name + "-" + ts.Name + "-metrics", "-o", "jsonpath='{.metadata.labels.chart}'",
	}

	output, err := shell.Execute(".", "kubectl", args...)
	if err != nil {
		return err
	}
	if output != ts.getFullName() {
		return errors.New("There is no " + resource + " for the " + ts.Name + " chart. Expected:" + ts.getFullName() + ", Actual: " + output)
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("A " + resource + " will manage additional pods for metricsets querying internal service")

	return nil
}

func (ts *HelmChartTestSuite) willRetrieveSpecificMetrics(chartName string) error {
	kubeStateMetrics := "kube-state-metrics"
	args := []string{
		"get", "deployments", ts.Name + "-" + kubeStateMetrics, "-o", "jsonpath='{.metadata.name}'",
	}

	output, err := shell.Execute(".", "kubectl", args...)
	if err != nil {
		return err
	}
	if output != ts.getKubeStateMetricsName() {
		return errors.New("There is no " + kubeStateMetrics + " Deployment for the " + ts.Name + " chart. Expected:" + ts.getKubeStateMetricsName() + ", Actual: " + output)
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("A " + kubeStateMetrics + " chart will retrieve specific Kubernetes metrics")

	return nil
}

func (ts *HelmChartTestSuite) aResourceContainsTheContent(resource string, content string) error {
	lowerResource := strings.ToLower(resource)
	escapedContent := strings.ReplaceAll(content, ".", `\.`)

	args := []string{
		"get", lowerResource + "s", ts.getResourceName(resource), "-o", `jsonpath="{.data['` + escapedContent + `']}"`,
	}

	output, err := shell.Execute(".", "kubectl", args...)
	if err != nil {
		return err
	}
	if output == "" {
		return errors.New("There is no " + resource + " for the " + ts.Name + " chart including " + content)
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("A " + resource + " resource contains the " + content + " content")

	return nil
}

func (ts *HelmChartTestSuite) aResourceManagesRBAC(resource string) error {
	lowerResource := strings.ToLower(resource)

	args := []string{
		"get", lowerResource + "s", ts.getResourceName(resource), "-o", `jsonpath="'{.metadata.labels.chart}'"`,
	}

	output, err := shell.Execute(".", "kubectl", args...)
	if err != nil {
		return err
	}
	if output == "" {
		return errors.New("There is no " + resource + " for the " + ts.Name + " chart")
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("A " + resource + " resource manages K8S RBAC")

	return nil
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
		testSuite.createCluster()
	})
	s.BeforeScenario(func(interface{}) {
		log.Info("Before Helm scenario...")
	})
	s.AfterSuite(func() {
		log.Debug("After Suite...")
		testSuite.destroyCluster()
	})
	s.AfterScenario(func(interface{}, error) {
		log.Debug("After Helm scenario...")
		testSuite.delete()
	})
}
