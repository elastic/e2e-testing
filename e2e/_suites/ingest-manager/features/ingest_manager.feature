@ingest
Feature: Ingest Manager
  Scenarios for the ingest manager application, considering the deployment, start, stop, enrollment, un-enrollment and enrollment of an agent.

@enroll
Scenario: Deploying an agent
  Given the "Fleet" Kibana setup has been executed
  When an agent is deployed to Fleet
  Then the agent is listed in Fleet as online
    And system package dashboards are listed in Fleet
    And there is data in the index

@start-agent
Scenario: Starting the agent starts backend processes
  When an agent is deployed to Fleet
  Then "filebeat" is "started" on the host
    And "metricbeat" is "started" on the host

@stop-agent
Scenario: Stopping the agent stops backend processes
  Given an agent is deployed to Fleet
  When the agent is "stopped" on the host
  Then "filebeat" is "stopped" on the host
    And "metricbeat" is "stopped" on the host

@unenroll
Scenario: Un-enrolling an agent
  Given an agent is deployed to Fleet
  When the agent is un-enrolled
  Then the agent is not listed as online in Fleet
    And there is no data in the index

@reenroll
Scenario: Re-enrolling an agent
  Given an agent is enrolled
    And the agent is un-enrolled
    And "the agent" is "stopped" on the host
  When the agent is re-enrolled on the host
    And "the agent" is "started" on the host
  Then "the agent" is listed in Fleet as online
    And there is data in the index

@revoke-token
Scenario: Revoking the enrollment token for an agent
  Given an agent is enrolled
  When the enrollment token is revoked
    And there is data in the index
  Then the agent is un-enrolled
    And "the agent" is "stopped" on the host
    And the agent cannot be re-enrolled with the same command
