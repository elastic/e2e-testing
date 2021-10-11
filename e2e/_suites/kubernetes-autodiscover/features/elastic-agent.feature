@kubernetes-autodiscover
@elastic-agent
Feature: elastic-agent standalone
  Use Kubernetes autodiscover features in elastic-agent standalone to collect logs and metrics

Scenario: Logs collection from running pod
  Given "elastic-agent" is running with "logs generic"
   When "a pod" is deployed
   Then "elastic-agent" collects events with "kubernetes.pod.name:a-pod"

Scenario: Logs collection from a pod with an init container
  Given "elastic-agent" is running with "logs generic init"
   When "a pod" is deployed with "init container"
   Then "elastic-agent" collects events with "kubernetes.container.name:init-container"
    And "elastic-agent" collects events with "kubernetes.container.name:container-in-pod"

# This scenario explicitly waits for 60 seconds before doing checks
# to be sure that at least one job has been executed.
Scenario: Logs collection from short-living cronjobs
  Given "elastic-agent" is running with "logs generic"
    And "a short-living cronjob" is deployed
   When "60s" have passed
   Then "elastic-agent" collects events with "kubernetes.container.name:cronjob-container"

Scenario: Logs collection from failing pod
  Given "elastic-agent" is running with "logs generic failing"
   When "a failing pod" is deployed
   Then "elastic-agent" collects events with "kubernetes.pod.name:a-failing-pod"

Scenario: Metrics collection configured from targeted Redis Pod
  Given "elastic-agent" is running with "redis info"
   When "redis" is running
   Then "elastic-agent" collects events with "kubernetes.pod.name:redis"