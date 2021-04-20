@beats
Feature: Beats
  Use Kubernetes autodiscover features in Beats to monitor pods

Scenario: Pod is started
  Given a cluster is available
    And configuration for "filebeat" has "hints enabled"
    And "filebeat" is deployed
  When "a pod" is deployed
  Then "filebeat" collects events

Scenario: Pod is deleted
  Given a cluster is available
    And configuration for "filebeat" has "hints enabled"
    And configuration for "metricbeat" has "hints enabled"
    And "filebeat" is deployed
    And "metricbeat" is deployed
    And "a pod" is deployed
  When "a pod" is deleted
  Then "filebeat" stop collecting events
    And "metricbeat" stop collecting events

Scenario: Pod is failing
  Given a cluster is available
    And configuration for "filebeat" has "hints enabled"
    And "filebeat" is deployed
    And "a failing pod" is deployed
  When "a failing pod" is deployed
  Then "filebeat" collects events

Scenario: Short-living cronjob
  Given a cluster is available
    And configuration for "filebeat" has "hints enabled"
    And "filebeat" is deployed
   When "a short-living cronjob" is deployed
   Then "filebeat" collects events
