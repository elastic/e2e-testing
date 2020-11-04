@apm-server
Feature: APM Server
  The Helm chart is following product recommended configuration for Kubernetes

Scenario: The APM Server chart will create recommended K8S resources
  Given a cluster is running
  When the "apm-server" Elastic's helm chart is installed
  Then a "Deployment" will manage the pods
    And a "Service" will expose the pods as network services internal to the k8s cluster
    And a "ConfigMap" resource contains the "apm-server.yml" key
    And a "ServiceAccount" resource manages RBAC
    And a "ClusterRole" resource manages RBAC
    And a "ClusterRoleBinding" resource manages RBAC
    And resource "limits" are applied
    And resource "requests" are applied
    And the "RollingUpdate" strategy can be used for "Deployment" during updates
