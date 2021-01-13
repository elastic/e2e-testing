// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/cli/config"
	k8s "github.com/elastic/e2e-testing/cli/services"
	shell "github.com/elastic/e2e-testing/cli/shell"
	"github.com/elastic/e2e-testing/e2e"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages-go/v10"
	log "github.com/sirupsen/logrus"
)

// developerMode tears down the backend services (the k8s cluster)
// after a test suite. This is the desired behavior, but when developing, we maybe want to keep
// them running to speed up the development cycle.
// It can be overriden by the DEVELOPER_MODE env var
var developerMode = false

var helm k8s.HelmManager

// timeoutFactor a multiplier for the max timeout when doing backoff retries.
// It can be overriden by TIMEOUT_FACTOR env var
var timeoutFactor = 2

//nolint:unused
var kubectl k8s.Kubectl

// helmVersion represents the default version used for Helm
var helmVersion = "3.x"

// helmChartVersion represents the default version used for the Elastic Helm charts
var helmChartVersion = "7.10.0"

// kubernetesVersion represents the default version used for Kubernetes
var kubernetesVersion = "1.18.2"

func init() {
	config.Init()

	developerMode = shell.GetEnvBool("DEVELOPER_MODE")
	if developerMode {
		log.Info("Running in Developer mode 💻: runtime dependencies between different test runs will be reused to speed up dev cycle")
	}

	helmVersion = shell.GetEnv("HELM_VERSION", helmVersion)
	helmChartVersion = shell.GetEnv("HELM_CHART_VERSION", helmChartVersion)
	kubernetesVersion = shell.GetEnv("HELM_KUBERNETES_VERSION", kubernetesVersion)
	timeoutFactor = shell.GetEnvInteger("TIMEOUT_FACTOR", timeoutFactor)

	h, err := k8s.HelmFactory(helmVersion)
	if err != nil {
		log.Fatalf("Helm could not be initialised: %v", err)
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
		log.WithField("error", err).Error("Could not check the status of the cluster.")
	}
	if output != ts.ClusterName {
		return fmt.Errorf("The cluster is not running")
	}

	log.WithFields(log.Fields{
		"output": output,
	}).Debug("Cluster is running")
	return nil
}

func (ts *HelmChartTestSuite) addElasticRepo() error {
	err := helm.AddRepo("elastic", "https://helm.elastic.co")
	if err != nil {
		log.WithField("error", err).Error("Could not add Elastic Helm repo")
	}
	return err
}

func (ts *HelmChartTestSuite) aResourceContainsTheKey(resource string, key string) error {
	lowerResource := strings.ToLower(resource)
	escapedKey := strings.ReplaceAll(key, ".", `\.`)

	output, err := kubectl.Run("get", lowerResource, ts.getResourceName(resource), "-o", `jsonpath="{.data['`+escapedKey+`']}"`)
	if err != nil {
		return err
	}
	if output == "" {
		return fmt.Errorf("There is no %s for the %s chart including %s", resource, ts.Name, key)
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("A " + resource + " resource contains the " + key + " key")

	return nil
}

func (ts *HelmChartTestSuite) aResourceManagesRBAC(resource string) error {
	lowerResource := strings.ToLower(resource)

	output, err := kubectl.Run("get", lowerResource, ts.getResourceName(resource), "-o", `jsonpath="'{.metadata.labels.chart}'"`)
	if err != nil {
		return err
	}
	if output == "" {
		return fmt.Errorf("There is no %s for the %s chart", resource, ts.Name)
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("A " + resource + " resource manages K8S RBAC")

	return nil
}

func (ts *HelmChartTestSuite) aResourceWillExposePods(resourceType string) error {
	selector, err := kubectl.GetResourceSelector("deployment", ts.Name+"-"+ts.Name)
	if err != nil {
		return err
	}

	maxTimeout := time.Duration(timeoutFactor) * time.Minute

	exp := e2e.GetExponentialBackOff(maxTimeout)
	retryCount := 1

	checkEndpointsFn := func() error {
		output, err := kubectl.GetStringResourcesBySelector("endpoints", selector)
		if err != nil {
			log.WithFields(log.Fields{
				"elapsedTime": exp.GetElapsedTime(),
				"error":       err,
				"selector":    selector,
				"resource":    "endpoints",
				"retry":       retryCount,
			}).Warn("Could not inspect resource with kubectl")

			retryCount++

			return err
		}

		jsonParsed, err := gabs.ParseJSON([]byte(output))
		if err != nil {
			log.WithFields(log.Fields{
				"elapsedTime": exp.GetElapsedTime(),
				"error":       err,
				"output":      output,
				"selector":    selector,
				"resource":    "endpoints",
				"retry":       retryCount,
			}).Warn("Could not parse JSON")

			retryCount++

			return err
		}

		subsets := jsonParsed.Path("items.0.subsets")
		if len(subsets.Children()) == 0 {
			log.WithFields(log.Fields{
				"elapsedTime": exp.GetElapsedTime(),
				"resource":    "endpoints",
				"retry":       retryCount,
				"selector":    selector,
			}).Warn("Endpoints not present yet")

			retryCount++

			return fmt.Errorf("Error there are no Endpoint subsets for the %s with the selector %s", resourceType, selector)
		}

		log.WithFields(log.Fields{
			"elapsedTime": exp.GetElapsedTime(),
			"resource":    "endpoints",
			"retry":       retryCount,
			"selector":    selector,
		}).Info("Endpoints found")

		return nil
	}

	err = backoff.Retry(checkEndpointsFn, exp)
	if err != nil {
		return err
	}

	return nil
}

func (ts *HelmChartTestSuite) aResourceWillManagePods(resourceType string) error {
	selector, err := kubectl.GetResourceSelector("deployment", ts.Name+"-"+ts.Name)
	if err != nil {
		return err
	}

	resources, err := ts.checkResources(resourceType, selector, 1)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"name":      ts.Name,
		"resources": resources,
	}).Tracef("Checking the %s pods", resourceType)

	return nil
}

func (ts *HelmChartTestSuite) checkResources(resourceType, selector string, min int) ([]interface{}, error) {
	resources, err := kubectl.GetResourcesBySelector(resourceType, selector)
	if err != nil {
		return nil, err
	}

	items := resources["items"].([]interface{})
	if len(items) < min {
		return nil, fmt.Errorf("Error there are not %d %s for resource %s/%s-%s with the selector %s", min, resourceType, resourceType, ts.Name, ts.Name, selector)
	}

	log.WithFields(log.Fields{
		"name":  ts.Name,
		"items": items,
	}).Tracef("Checking for %d %s with selector %s", min, resourceType, selector)

	return items, nil
}

func (ts *HelmChartTestSuite) createCluster(k8sVersion string) error {
	args := []string{"create", "cluster", "--name", ts.ClusterName, "--image", "kindest/node:v" + k8sVersion}

	log.Trace("Creating cluster with kind")
	output, err := shell.Execute(".", "kind", args...)
	if err != nil {
		log.WithField("error", err).Error("Could not create the cluster")
		return err
	}
	log.WithFields(log.Fields{
		"cluster":    ts.ClusterName,
		"k8sVersion": k8sVersion,
		"output":     output,
	}).Info("Cluster created")

	return nil
}

func (ts *HelmChartTestSuite) deleteChart() {
	err := helm.DeleteChart(ts.Name)
	if err != nil {
		log.WithFields(log.Fields{
			"chart": ts.Name,
		}).Error("Could not delete chart")
	}
}

func (ts *HelmChartTestSuite) destroyCluster() error {
	args := []string{"delete", "cluster", "--name", ts.ClusterName}

	log.Trace("Deleting cluster")
	output, err := shell.Execute(".", "kind", args...)
	if err != nil {
		log.WithField("error", err).Error("Could not destroy the cluster")
		return err
	}
	log.WithFields(log.Fields{
		"output":  output,
		"cluster": ts.ClusterName,
	}).Debug("Cluster destroyed")
	return nil
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

// getPodName returns the name used in the app selector, in lowercase
func (ts *HelmChartTestSuite) getPodName() string {
	if ts.Name == "apm-server" {
		return strings.ToLower(ts.Name)
	}

	return strings.ToLower(ts.Name + "-" + ts.Name)
}

// getResourceName returns the name of the service, in lowercase, based on the k8s resource
func (ts *HelmChartTestSuite) getResourceName(resource string) string {
	if resource == k8s.ResourceTypes.ClusterRole {
		return strings.ToLower(ts.Name + "-" + ts.Name + "-cluster-role")
	} else if resource == k8s.ResourceTypes.ClusterRoleBinding {
		return strings.ToLower(ts.Name + "-" + ts.Name + "-cluster-role-binding")
	} else if resource == k8s.ResourceTypes.ConfigMap {
		if ts.Name == "metricbeat" {
			return strings.ToLower(ts.Name + "-" + ts.Name + "-daemonset-config")
		}
		return strings.ToLower(ts.Name + "-" + ts.Name + "-config")
	} else if resource == k8s.ResourceTypes.Daemonset {
		return strings.ToLower(ts.Name + "-" + ts.Name)
	} else if resource == k8s.ResourceTypes.Deployment {
		if ts.Name == "metricbeat" {
			return strings.ToLower(ts.Name + "-" + ts.Name + "-metrics")
		}
		return strings.ToLower(ts.Name + "-" + ts.Name)
	} else if resource == k8s.ResourceTypes.ServiceAccount {
		return strings.ToLower(ts.Name + "-" + ts.Name)
	}

	return ""
}

func (ts *HelmChartTestSuite) install(chart string) error {
	ts.Name = chart

	elasticChart := "elastic/" + ts.Name

	flags := []string{}
	if chart == "elasticsearch" {
		// Rancher Local Path Provisioner and local-path storage class for Elasticsearch volumes
		_, err := kubectl.Run("apply", "-f", "https://raw.githubusercontent.com/rancher/local-path-provisioner/master/deploy/local-path-storage.yaml")
		if err != nil {
			log.Errorf("Could not apply Rancher Local Path Provisioner: %v", err)
			return err
		}
		log.WithFields(log.Fields{
			"chart": ts.Name,
		}).Info("Rancher Local Path Provisioner and local-path storage class for Elasticsearch volumes installed")

		maxTimeout := timeoutFactor * 100

		log.Debug("Applying workaround to use Rancher's local-path storage class for Elasticsearch volumes")
		flags = []string{"--wait", fmt.Sprintf("--timeout=%ds", maxTimeout), "--values", "https://raw.githubusercontent.com/elastic/helm-charts/master/elasticsearch/examples/kubernetes-kind/values.yaml"}
	}

	return helm.InstallChart(ts.Name, elasticChart, ts.Version, flags)
}

func (ts *HelmChartTestSuite) installRuntimeDependencies(dependencies ...string) error {
	for _, dependency := range dependencies {
		// Install Elasticsearch
		err := ts.install(dependency)
		if err != nil {
			log.WithFields(log.Fields{
				"dependency": dependency,
				"error":      err,
			}).Error("Could not install runtime dependency")
			return err
		}
	}

	return nil
}

func (ts *HelmChartTestSuite) podsManagedByDaemonSet() error {
	output, err := kubectl.Run("get", "daemonset", "--namespace=default", "-l", "app="+ts.Name+"-"+ts.Name, "-o", "jsonpath='{.items[0].metadata.labels.chart}'")
	if err != nil {
		return err
	}
	if output != ts.getFullName() {
		return fmt.Errorf("There is no DaemonSet for the %s chart. Expected: %s, Actual: %s", ts.Name, ts.getFullName(), output)
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("A pod will be deployed on each node of the cluster by a DaemonSet")

	return nil
}

func (ts *HelmChartTestSuite) resourceConstraintsAreApplied(constraint string) error {
	output, err := kubectl.Run("get", "pods", "-l", "app="+ts.getPodName(), "-o", "jsonpath='{.items[0].spec.containers[0].resources."+constraint+"}'")
	if err != nil {
		return err
	}
	if output == "" {
		return fmt.Errorf("Resource %s constraint for the %s chart is not applied. Actual: %s", constraint, ts.getFullName(), output)
	}

	log.WithFields(log.Fields{
		"constraint": constraint,
		"name":       ts.Name,
		"output":     output,
	}).Debug("Resource" + constraint + " is applied")

	return nil
}

func (ts *HelmChartTestSuite) resourceWillManageAdditionalPodsForMetricsets(resource string) error {
	lowerResource := strings.ToLower(resource)

	output, err := kubectl.Run("get", lowerResource, ts.getResourceName(resource), "-o", "jsonpath='{.metadata.labels.chart}'")
	if err != nil {
		return err
	}
	if output != ts.getFullName() {
		return fmt.Errorf("There is no %s for the %s chart. Expected: %s, Actual: %s", resource, ts.Name, ts.getFullName(), output)
	}

	log.WithFields(log.Fields{
		"output": output,
		"name":   ts.Name,
	}).Debug("A " + resource + " will manage additional pods for metricsets querying internal service")

	return nil
}

func (ts *HelmChartTestSuite) strategyCanBeUsedDuringUpdates(strategy string) error {
	return ts.strategyCanBeUsedForResourceDuringUpdates(strategy, k8s.ResourceTypes.Daemonset)
}

func (ts *HelmChartTestSuite) strategyCanBeUsedForResourceDuringUpdates(strategy string, resource string) error {
	lowerResource := strings.ToLower(resource)
	strategyKey := "strategy"
	name := ts.getResourceName(resource)

	if resource == k8s.ResourceTypes.Daemonset {
		strategyKey = "updateStrategy"
	}

	output, err := kubectl.Run("get", lowerResource, name, "-o", `go-template={{.spec.`+strategyKey+`.type}}`)
	if err != nil {
		return err
	}
	if output != strategy {
		return fmt.Errorf("There is no %s strategy to be used for %s on updates. Actual: %s", strategy, resource, output)
	}

	log.WithFields(log.Fields{
		"strategy": strategy,
		"resource": resource,
		"name":     name,
	}).Debug("The strategy can be used for resource during updates")

	return nil
}

func (ts *HelmChartTestSuite) volumeMountedWithNoSubpath(name string, mountPath string) error {
	return ts.volumeMountedWithSubpath(name, mountPath, "")
}

func (ts *HelmChartTestSuite) volumeMountedWithSubpath(name string, mountPath string, subPath string) error {

	getMountValues := func(key string) ([]string, error) {
		// build the arguments for capturing the volume mounts
		output, err := kubectl.Run("get", "pods", "-l", "app="+ts.getPodName(), "-o", `jsonpath="{.items[0].spec.containers[0].volumeMounts[*]['`+key+`']}"`)
		if err != nil {
			return []string{}, err
		}
		output = strings.Trim(output, "\"") // remove enclosing double quotes

		return strings.Split(output, " "), nil
	}

	// get volumeMounts names
	names, err := getMountValues("name")
	if err != nil {
		return err
	}

	// Find returns the smallest index i at which x == a[i],
	// or len(a) if there is no such index.
	find := func(a []string, x string) int {
		for i, n := range a {
			if x == n {
				return i
			}
		}
		return len(a)
	}

	index := find(names, name)
	if index == len(names) {
		return fmt.Errorf("The mounted volume '%s' could not be found: %v", name, names)
	}

	// get mounts paths
	mountPaths, err := getMountValues("mountPath")
	if err != nil {
		return err
	}

	if mountPath != mountPaths[index] {
		return fmt.Errorf("The mounted volume for '%s' is not %s. Actual: %s", name, mountPath, mountPaths[index])
	}

	if subPath != "" {
		// get subpaths
		subPaths, err := getMountValues("subPath")
		if err != nil {
			return err
		}

		if subPath != subPaths[index] {
			return fmt.Errorf("The subPath for '%s' is not %s. Actual: %s", name, subPath, subPaths[index])
		}
	}

	log.WithFields(log.Fields{
		"name":      name,
		"mountPath": mountPath,
		"subPath":   subPath,
	}).Debug("The volumePath was found")

	return nil
}

func (ts *HelmChartTestSuite) willRetrieveSpecificMetrics(chartName string) error {
	kubeStateMetrics := "kube-state-metrics"

	output, err := kubectl.Run("get", "deployment", ts.Name+"-"+kubeStateMetrics, "-o", "jsonpath='{.metadata.name}'")
	if err != nil {
		return err
	}
	if output != ts.getKubeStateMetricsName() {
		return fmt.Errorf("There is no %s Deployment for the %s chart. Expected: %s, Actual: %s", kubeStateMetrics, ts.Name, ts.getKubeStateMetricsName(), output)
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
		KubernetesVersion: kubernetesVersion,
		Version:           helmChartVersion,
	}

	s.Step(`^a cluster is running$`, testSuite.aClusterIsRunning)
	s.Step(`^the "([^"]*)" Elastic\'s helm chart is installed$`, testSuite.elasticsHelmChartIsInstalled)
	s.Step(`^a pod will be deployed on each node of the cluster by a DaemonSet$`, testSuite.podsManagedByDaemonSet)
	s.Step(`^a "([^"]*)" will manage additional pods for metricsets querying internal services$`, testSuite.resourceWillManageAdditionalPodsForMetricsets)
	s.Step(`^a "([^"]*)" chart will retrieve specific Kubernetes metrics$`, testSuite.willRetrieveSpecificMetrics)
	s.Step(`^a "([^"]*)" resource contains the "([^"]*)" key$`, testSuite.aResourceContainsTheKey)
	s.Step(`^a "([^"]*)" resource manages RBAC$`, testSuite.aResourceManagesRBAC)
	s.Step(`^the "([^"]*)" volume is mounted at "([^"]*)" with subpath "([^"]*)"$`, testSuite.volumeMountedWithSubpath)
	s.Step(`^the "([^"]*)" volume is mounted at "([^"]*)" with no subpath$`, testSuite.volumeMountedWithNoSubpath)
	s.Step(`^the "([^"]*)" strategy can be used during updates$`, testSuite.strategyCanBeUsedDuringUpdates)
	s.Step(`^the "([^"]*)" strategy can be used for "([^"]*)" during updates$`, testSuite.strategyCanBeUsedForResourceDuringUpdates)
	s.Step(`^resource "([^"]*)" are applied$`, testSuite.resourceConstraintsAreApplied)

	s.Step(`^a "([^"]*)" will manage the pods$`, testSuite.aResourceWillManagePods)
	s.Step(`^a "([^"]*)" will expose the pods as network services internal to the k8s cluster$`, testSuite.aResourceWillExposePods)

	s.BeforeSuite(func() {
		log.Trace("Before Suite...")
		toolsAreInstalled()

		err := testSuite.createCluster(testSuite.KubernetesVersion)
		if err != nil {
			return
		}
		err = testSuite.addElasticRepo()
		if err != nil {
			return
		}
		err = testSuite.installRuntimeDependencies("elasticsearch")
		if err != nil {
			return
		}
	})
	s.BeforeScenario(func(*messages.Pickle) {
		log.Trace("Before Helm scenario...")
	})
	s.AfterSuite(func() {
		if !developerMode {
			log.Trace("After Suite...")
			err := testSuite.destroyCluster()
			if err != nil {
				return
			}
		}
	})
	s.AfterScenario(func(*messages.Pickle, error) {
		log.Trace("After Helm scenario...")
		testSuite.deleteChart()
	})
}

//nolint:unused
func toolsAreInstalled() {
	binaries := []string{
		"kind",
		"kubectl",
		"helm",
	}

	shell.CheckInstalledSoftware(binaries)
}
