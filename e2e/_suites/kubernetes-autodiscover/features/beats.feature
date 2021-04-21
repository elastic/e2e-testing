@beats
Feature: Beats
  Use Kubernetes autodiscover features in Beats to monitor pods

Scenario: Pod is started
  Given configuration for "filebeat" has "hints enabled"
    And "filebeat" is running
   When "a pod" is deployed
   Then "filebeat" collects events with "kubernetes.pod.name:a-pod"

Scenario: Pod is deleted
  Given configuration for "filebeat" has "hints enabled"
    And "filebeat" is running
    And "a pod" is deployed
    And "filebeat" collects events with "kubernetes.pod.name:a-pod"
   When "a pod" is deleted
   Then "filebeat" stops collecting events

Scenario: Pod is failing
  Given configuration for "filebeat" has "hints enabled"
    And "filebeat" is running
   When "a failing pod" is deployed
   Then "filebeat" collects events with "kubernetes.pod.name:a-failing-pod"

# This scenario explicitly waits for 60 seconds before doing checks
# to be sure that at least one job has been executed.
Scenario: Short-living cronjob
  Given configuration for "filebeat" has "hints enabled"
    And "filebeat" is running
   When "a short-living cronjob" is deployed
    And "60s" have passed
   Then "filebeat" collects events with "kubernetes.container.name:cronjob-container"

Scenario: Metrics hints with named ports
  Given configuration for "metricbeat" has "hints enabled"
    And configuration for "a pod" has "metrics annotations with named port"
    And "metricbeat" is running
   When "a pod" is deployed
   Then "metricbeat" collects events with "kubernetes.pod.name:a-pod"

Scenario: Monitor hints with named ports
  Given configuration for "heartbeat" has "hints enabled"
    And configuration for "a service" has "monitor annotations with named port"
    And "heartbeat" is running
   When "a service" is deployed
   Then "heartbeat" collects events with "kubernetes.service.name:a-service"
