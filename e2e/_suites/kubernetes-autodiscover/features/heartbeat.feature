@kubernetes-autodiscover
@heartbeat
Feature: Heartbeat
  Use Kubernetes autodiscover features in Heartbeat to monitor pods, services and nodes.

Scenario: Monitor pod availability using hints with named ports
  Given "heartbeat" is running with "hints enabled for pods"
   When "redis" is deployed with "monitor annotations with named port"
   Then "heartbeat" collects events with "kubernetes.pod.name:redis"
    And "heartbeat" collects events with "monitor.status:up"

Scenario: Monitor service availability using hints
  Given "heartbeat" is running with "hints enabled for services"
    And "redis service" is deployed with "monitor annotations"
   When "redis" is running
   Then "heartbeat" collects events with "kubernetes.service.name:redis"
    And "heartbeat" collects events with "monitor.status:up"
    And "heartbeat" does not collect events with "monitor.status:down" during "20s"

# A service without backend pods should be reported as down.
Scenario: Monitor service unavailability using hints
  Given "heartbeat" is running with "hints enabled for services"
   When "redis service" is deployed with "monitor annotations"
   Then "heartbeat" collects events with "kubernetes.service.name:redis"
    And "heartbeat" collects events with "monitor.status:down"
    And "heartbeat" does not collect events with "monitor.status:up" during "20s"

Scenario: Monitor nodes using hints
   When "heartbeat" is running with "hints enabled for nodes"
   Then "heartbeat" collects events with "url.port:10250"
    And "heartbeat" collects events with "monitor.status:up"
