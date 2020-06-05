@ingest
Feature: Enable Fleet and Deploy Agent basic end to end tests

@enroll
Scenario: Deploying an agent
  Given the "Fleet" Kibana setup has been executed
    And an agent is deployed to Fleet
  Then the agent is listed in Fleet as online
    And system package dashboards are listed in Fleet
    And new documents are inserted into Elasticsearch

@start-agent
Scenario: Starting the agent starts backend processes
  Given an agent is deployed to Fleet
  Then filebeat is started
    And metricbeat is started

@stop-agent
Scenario: Stopping the agent stops backend processes
  Given an agent is deployed to Fleet
  When the agent is stopped on the host
  Then filebeat is stopped
    And metricbeat is stopped

@unenroll
Scenario: Un-enrolling an agent
  Given an agent is deployed to Fleet
  When the agent is un-enrolled
  Then the agent is not listed as online in Fleet
    And no new documents are inserted into Elasticsearch

@reenroll
Scenario: Re-enrolling an agent
  Given an agent is enrolled
    And the agent is un-enrolled and stopped on the host
  When the agent is re-enrolled on the host
    And the agent is run on the host
  Then the agent is listed in Fleet as online
    And new documents are inserted into Elasticsearch

@revoke-token
Scenario: Revoking the enrollment token for an agent
  Given an agent is enrolled
  When the enrollment token is revoked
    Then new documents are inserted into Elasticsearch
  And the agent is un-enrolled and stopped on the host
  Then the agent cannot be re-enrolled with the same command
