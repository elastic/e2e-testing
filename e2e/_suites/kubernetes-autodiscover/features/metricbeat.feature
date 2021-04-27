@kubernetes-autodiscover
@metricbeat
Feature: Metricbeat
  Use Kubernetes autodiscover features in Metricbeat to discover services and
  collect metrics from them.

Scenario: Metrics collection stops when the pod is stopped
  Given "metricbeat" is running with "hints enabled"
    And "redis" is running with "metrics annotations"
    And "metricbeat" collects events with "kubernetes.pod.name:redis"
   When "redis" is deleted
   Then "metricbeat" does not collect events with "kubernetes.pod.name:redis" during "30s"

Scenario: Metrics collection configured with hints with named ports
  Given "metricbeat" is running with "hints enabled"
   When "redis" is running with "metrics annotations with named port"
   Then "metricbeat" collects events with "kubernetes.pod.name:redis"
