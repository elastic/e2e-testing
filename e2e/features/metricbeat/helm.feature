@helm
@k8s
@metricbeat
Feature: The Helm chart is following product recommended configuration for Kubernetes

Scenario: The Metricbeat chart will create recommended K8S resources
  Given a cluster is running
  When the "metricbeat" Elastic's helm chart is installed
  Then a pod will be deployed on each node of the cluster by a DaemonSet
    And a "Deployment" will manage additional pods for metricsets querying internal services
    And a "kube-state-metrics" chart will retrieve specific Kubernetes metrics
    And a "ConfigMap" resource contains the "metricbeat.yml" key
    And a "ConfigMap" resource contains the "kube-state-metrics-metricbeat.yml" key
    And a "ServiceAccount" resource manages RBAC
    And a "ClusterRole" resource manages RBAC
    And a "ClusterRoleBinding" resource manages RBAC
    And resource "limits" are applied
    And resource "requests" are applied
    And the "RollingUpdate" strategy can be used for "Deployment" during updates
    And the "RollingUpdate" strategy can be used for "Daemonset" during updates
    And the "data" volume is mounted at "/usr/share/metricbeat/data" with no subpath
    And the "varlibdockercontainers" volume is mounted at "/var/lib/docker/containers" with no subpath
    And the "varrundockersock" volume is mounted at "/var/run/docker.sock" with no subpath
    And the "proc" volume is mounted at "/hostfs/proc" with no subpath
    And the "cgroup" volume is mounted at "/hostfs/sys/fs/cgroup" with no subpath
