@kubernetes
@autodiscover
@beats
Feature: Log collection from Kubernetes pods.
  Use Kubernetes autodiscover features in Filebeat to collect logs.

Scenario: Logs collection from running pod
  Given "filebeat" is running with "hints enabled"
   When "a pod" is deployed
   Then "filebeat" collects events with "kubernetes.pod.name:a-pod"

Scenario: Logs collection from failing pod
  Given "filebeat" is running with "hints enabled"
   When "a failing pod" is deployed
   Then "filebeat" collects events with "kubernetes.pod.name:a-failing-pod"

# This scenario explicitly waits for 60 seconds before doing checks
# to be sure that at least one job has been executed.
Scenario: Logs collection from short-living cronjobs
  Given "filebeat" is running with "hints enabled"
   When "a short-living cronjob" is deployed
    And "60s" have passed
   Then "filebeat" collects events with "kubernetes.container.name:cronjob-container"

Scenario: Logs collection from a pod with an init container
  Given "filebeat" is running with "hints enabled"
   When "a pod" is deployed with "init container"
   Then "filebeat" collects events with "kubernetes.container.name:init-container"
    And "filebeat" collects events with "kubernetes.container.name:container-in-pod"

# Ephemeral containers need to be explicitly enabled in the API server with the
# `EphemeralContainers` feature flag.
Scenario: Logs collection from a pod with an ephemeral container
  Given "filebeat" is running with "hints enabled"
    And "a pod" is deployed
    And "filebeat" collects events with "kubernetes.container.name:container-in-pod"
   When an ephemeral container is started in "a pod"
   Then "filebeat" collects events with "kubernetes.container.name:ephemeral-container"
