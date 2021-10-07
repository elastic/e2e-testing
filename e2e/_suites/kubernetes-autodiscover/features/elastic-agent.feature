@kubernetes-autodiscover
@elastic-agent
Feature: elastic-agent standalone
  Use Kubernetes autodiscover features in elastic-agent standalone to collect logs and metrics

Scenario: Logs collection from running pod
  Given "elastic-agent" is running
   When "a pod" is deployed
   Then "elastic-agent" collects events with "kubernetes.pod.name:a-pod"