package e2e

import (
	"errors"
	"os"
	"runtime"
	"strings"

	services "github.com/elastic/metricbeat-tests-poc/cli/services"
	shell "github.com/elastic/metricbeat-tests-poc/cli/shell"

	"github.com/cucumber/godog"
	log "github.com/sirupsen/logrus"
)

var helm services.HelmManager

func init() {
	h, err := services.HelmFactory("3.1.1")
	if err != nil {
		log.Fatal(err)
	}
	helm = h
}

// HelmChartTestSuite represents a test suite for a helm chart
//nolint:unused
type HelmChartTestSuite struct {
	Name    string // the name of the chart
	Version string // the helm chart version for the test
}

func (ts *HelmChartTestSuite) aClusterIsRunning() error {
	args := []string{"status"}

	output, err := shell.Execute(".", "minikube", args...)
	if err != nil {
		log.Fatal("Could not check the status of the cluster. Aborting")
	}
	log.WithFields(log.Fields{
		"output": output,
	}).Debug("Cluster is running")
	return nil
}

func (ts *HelmChartTestSuite) addElasticRepo() {
	err := helm.AddRepo("elastic", "https://helm.elastic.co")
	if err != nil {
		log.Fatal("Could not add Elastic Helm repo. Aborting")
	}
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

func (ts *HelmChartTestSuite) createCluster() {
	flags := ""
	if runtime.GOOS == "linux" {
		// Minikube also supports a --vm-driver=none option that runs the Kubernetes components
		// on the host and not in a VM. Using this driver requires Docker and a Linux environment
		// but not a hypervisor.
		flags = "--vm-driver=none"
	}

	args := []string{"start", flags}

	log.Debug("Creating cluster with minikube")
	output, err := shell.Execute(".", "minikube", args...)
	if err != nil {
		log.Fatal("Could not create the cluster. Aborting")
	}
	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("Cluster created")
}

func (ts *HelmChartTestSuite) deleteChart() error {
	return helm.DeleteChart(ts.Name)
}

func (ts *HelmChartTestSuite) destroyCluster() {
	args := []string{"delete"}

	log.Debug("Deleting cluster")
	output, err := shell.Execute(".", "minikube", args...)
	if err != nil {
		log.Fatal("Could not destroy the cluster. Aborting")
	}
	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("Cluster destroyed")
}

func (ts *HelmChartTestSuite) elasticsHelmChartIsInstalled(chart string) error {
	return ts.install(chart)
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

func (ts *HelmChartTestSuite) install(chart string) error {
	ts.Name = chart

	elasticChart := "elastic/" + ts.Name

	return helm.InstallChart(ts.Name, elasticChart, ts.Version)
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

// HelmChartFeatureContext adds steps to the Godog test suite
//nolint:deadcode,unused
func HelmChartFeatureContext(s *godog.Suite) {
	testSuite := HelmChartTestSuite{
		Version: "7.6.1",
	}

	if value, exists := os.LookupEnv("HELM_CHART_VERSION"); exists {
		testSuite.Version = value
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
		toolsAreInstalled()

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
		testSuite.deleteChart()
	})
}

func toolsAreInstalled() {
	binaries := []string{
		"kubectl",
		"helm",
		"minikube",
	}

	shell.CheckInstalledSoftware(binaries)
}
