@helm
@k8s
@metricbeat
Feature: The Helm chart is following product recommended configuration for Kubernetes

Background:
  Given tools are installed

Scenario: The Metricbeat chart will create recommended K8S resources
  Given a cluster is running
  When the "metricbeat" Elastic's helm chart is installed
  Then a pod will be deployed on each node of the cluster by a DaemonSet
    And a "Deployment" will manage additional pods for metricsets querying internal services
    And a "kube-state-metrics" chart will retrieve specific Kubernetes metrics
    And a "ConfigMap" resource contains the "metricbeat.yml" content
    And a "ConfigMap" resource contains the "kube-state-metrics-metricbeat.yml" content
    And a "ServiceAccount" resource manages RBAC
    And a "ClusterRole" resource manages RBAC
    And a "ClusterRoleBinding" resource manages RBAC