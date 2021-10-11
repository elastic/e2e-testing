@kubernetes-autodiscover
@elastic-agent
Feature: elastic-agent standalone
  Use Kubernetes autodiscover features in elastic-agent standalone to collect logs and metrics

Scenario: Logs collection from running pod
  Given "elastic-agent" is running with "logs generic"
   When "a pod" is deployed
   Then "elastic-agent" collects events with "kubernetes.pod.name:a-pod"

Scenario: Metrics collection configured from targeted Redis Pod
  Given "elastic-agent" is running with "redis info"
   When "redis" is running
   Then "elastic-agent" collects events with "kubernetes.pod.name:redis"