package e2e

import (
	"errors"
	"os"
	"strings"

	services "github.com/elastic/metricbeat-tests-poc/cli/services"
	shell "github.com/elastic/metricbeat-tests-poc/cli/shell"

	"github.com/cucumber/godog"
	log "github.com/sirupsen/logrus"
)

var helm services.HelmManager

func init() {
	helmVersion := "2.x"
	if value, exists := os.LookupEnv("HELM_VERSION"); exists {
		helmVersion = value
	}

	h, err := services.HelmFactory(helmVersion)
	if err != nil {
		log.Fatal(err)
	}
	helm = h
}

// HelmChartTestSuite represents a test suite for a helm chart
//nolint:unused
type HelmChartTestSuite struct {
	ClusterName       string // the name of the cluster
	KubernetesVersion string // the Kubernetes version for the test
	Name              string // the name of the chart
	Version           string // the helm chart version for the test
}

func (ts *HelmChartTestSuite) aClusterIsRunning() error {
	args := []string{"get", "clusters"}

	output, err := shell.Execute(".", "kind", args...)
	if err != nil {
		log.Fatalf("Could not check the status of the cluster. Aborting: %v", err)
	}
	if output != ts.ClusterName {
		return errors.New("The cluster is not running")
	}

	log.WithFields(log.Fields{
		"output": output,
	}).Debug("Cluster is running")
	return nil
}

func (ts *HelmChartTestSuite) addElasticRepo() {
	err := helm.AddRepo("elastic", "https://helm.elastic.co")
	if err != nil {
		log.Fatalf("Could not add Elastic Helm repo. Aborting: %v", err)
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

func (ts *HelmChartTestSuite) createCluster(k8sVersion string) {
	args := []string{"create", "cluster", "--name", ts.ClusterName, "--image", "kindest/node:v" + k8sVersion}

	log.Debug("Creating cluster with kind")
	output, err := shell.Execute(".", "kind", args...)
	if err != nil {
		log.Fatalf("Could not create the cluster. Aborting: %v", err)
	}
	log.WithFields(log.Fields{
		"cluster":    ts.ClusterName,
		"k8sVersion": k8sVersion,
		"output":     output,
	}).Debug("Cluster created")

	// initialise Helm after the cluster is created
	// For Helm v2.x.x we have to initialise Tiller
	// right after the k8s cluster
	err = helm.Init()
	if err != nil {
		log.Fatalf("Could not initiase Helm. Aborting: %v", err)
	}
}

func (ts *HelmChartTestSuite) deleteChart() error {
	return helm.DeleteChart(ts.Name)
}

func (ts *HelmChartTestSuite) destroyCluster() {
	args := []string{"delete", "cluster", "--name", ts.ClusterName}

	log.Debug("Deleting cluster")
	output, err := shell.Execute(".", "kind", args...)
	if err != nil {
		log.Fatalf("Could not destroy the cluster. Aborting: %v", err)
	}
	log.WithFields(log.Fields{
		"output":  output,
		"cluster": ts.ClusterName,
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
		ClusterName:       "helm-charts-test-suite",
		KubernetesVersion: "1.15.3",
		Version:           "7.6.1",
	}

	if value, exists := os.LookupEnv("HELM_CHART_VERSION"); exists {
		testSuite.Version = value
	}
	if value, exists := os.LookupEnv("KUBERNETES_VERSION"); exists {
		testSuite.KubernetesVersion = value
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

		testSuite.createCluster(testSuite.KubernetesVersion)
		testSuite.addElasticRepo()
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
		"kind",
		"kubectl",
		"helm",
	}

	shell.CheckInstalledSoftware(binaries)
}
