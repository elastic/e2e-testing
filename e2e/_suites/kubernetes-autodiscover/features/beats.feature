@kubernetes
@autodiscover
@beats
Feature: Beats
  Use Kubernetes autodiscover features in Beats to monitor pods

Scenario: Pod is started
  Given "filebeat" is running with "hints enabled"
   When "a pod" is deployed
   Then "filebeat" collects events with "kubernetes.pod.name:a-pod"

Scenario: Pod is deleted
  Given "metricbeat" is running with "hints enabled"
    And "redis" is deployed with "metrics annotations"
    And "metricbeat" collects events with "kubernetes.pod.name:redis"
   When "redis" is deleted
   Then "metricbeat" does not collect events with "kubernetes.pod.name:redis" during "30s"

Scenario: Pod is failing
  Given "filebeat" is running with "hints enabled"
   When "a failing pod" is deployed
   Then "filebeat" collects events with "kubernetes.pod.name:a-failing-pod"

# This scenario explicitly waits for 60 seconds before doing checks
# to be sure that at least one job has been executed.
Scenario: Short-living cronjob
  Given "filebeat" is running with "hints enabled"
   When "a short-living cronjob" is deployed
    And "60s" have passed
   Then "filebeat" collects events with "kubernetes.container.name:cronjob-container"

Scenario: Init container in a pod
  Given "filebeat" is running with "hints enabled"
   When "a pod" is deployed with "init container"
   Then "filebeat" collects events with "kubernetes.container.name:init-container"
    And "filebeat" collects events with "kubernetes.container.name:container-in-pod"

# Ephemeral containers need to be explicitly enabled in the API server with the
# `EphemeralContainers` feature flag.
Scenario: Ephemeral container in a pod
  Given "filebeat" is running with "hints enabled"
    And "a pod" is deployed
    And "filebeat" collects events with "kubernetes.container.name:container-in-pod"
   When an ephemeral container is started in "a pod"
   Then "filebeat" collects events with "kubernetes.container.name:ephemeral-container"

Scenario: Metrics hints with named ports
  Given "metricbeat" is running with "hints enabled"
   When "redis" is deployed with "metrics annotations with named port"
   Then "metricbeat" collects events with "kubernetes.pod.name:redis"

Scenario: Monitor hints for pods with named ports
  Given "heartbeat" is running with "hints enabled for pods"
   When "redis" is deployed with "monitor annotations with named port"
   Then "heartbeat" collects events with "kubernetes.pod.name:redis"
