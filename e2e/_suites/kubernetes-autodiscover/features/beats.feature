@beats
Feature: Beats
  Use Kubernetes autodiscover features in Beats to monitor pods

Scenario: Pod is started
  Given a cluster is available
    And configuration for "filebeat" has "hints enabled"
    And "filebeat" is deployed
  When "a pod" is deployed
  Then "filebeat" collects events for "a pod"

Scenario: Pod is deleted
  Given a cluster is available
    And configuration for "filebeat" has "hints enabled"
    And "filebeat" is deployed
    And "a pod" is deployed
    And "filebeat" collects events for "a pod"
  When "a pod" is deleted
  Then "filebeat" stops collecting events

Scenario: Pod is failing
  Given a cluster is available
    And configuration for "filebeat" has "hints enabled"
    And "filebeat" is deployed
  When "a failing pod" is deployed
  Then "filebeat" collects events for "a failing pod"

Scenario: Short-living cronjob
  Given a cluster is available
    And configuration for "filebeat" has "hints enabled"
    And "filebeat" is deployed
   When "a short-living cronjob" is deployed
   Then "filebeat" collects events for "a failing pod"
