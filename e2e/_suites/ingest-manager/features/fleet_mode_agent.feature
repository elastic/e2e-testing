@fleet_mode
Feature: Fleet Mode Agent
  Scenarios for the Agent in Fleet mode connecting to Ingest Manager application.

@enroll
Scenario: Deploying an agent
  Given the "Fleet" Kibana setup has been executed
  When an agent is deployed to Fleet
  Then the agent is listed in Fleet as online
    And the "filebeat" process is "started" on the host
    And the "metricbeat" process is "started" on the host
    And system package dashboards are listed in Fleet
    And there is new data in the index from agent

@stop-agent
Scenario: Stopping the agent stops backend processes
  Given an agent is deployed to Fleet
  When the "elastic-agent" process is "stopped" on the host
  Then the "filebeat" process is "stopped" on the host
    And the "metricbeat" process is "stopped" on the host
    And there is no new data in the index after agent shuts down

@unenroll
Scenario: Un-enrolling an agent
  Given an agent is deployed to Fleet
  When the agent is un-enrolled
  Then the agent is not listed as online in Fleet
    And there is no new data in the index after agent shuts down

@reenroll
Scenario: Re-enrolling an agent
  Given an agent is deployed to Fleet
    And the agent is un-enrolled
    And the "agent" process is "stopped" on the host
  When the agent is re-enrolled on the host
    And the "agent" process is "started" on the host
  Then the agent is listed in Fleet as online
    And there is new data in the index from agent

@revoke-token
Scenario: Revoking the enrollment token for an agent
  Given an agent is deployed to Fleet
    And the agent is un-enrolled
    And the "agent" process is "stopped" on the host
  When the enrollment token is revoked
  Then an attempt to enroll an agent with the old token fails

@package-added-to-default-config
Scenario: Execute packages api calls
  Given an agent is deployed to Fleet
    And the package list api returns successfully
  When the "Cisco" latest package version is installed successfull
    And a "Cisco" package datasource is added to the 'default' configuration
  Then the "default" configuration shows the "Cisco" datasource added

@new-agent-configuration
Scenario: Assign an Agent to a new configuration
  Given an agent is deployed to Fleet
    And the agent is listed in Fleet as online
  When a new configuration named "Test Fleet" is created
    And the Agent is assigned to the configuration "Test Fleet"
  Then a new enrollment token is created
    And there is new data in the index from agent

@new-configuration-new-package
Scenario: Add a new config and a new package and assign an agent
  Given an agent is deployed to Fleet
  When a new configuration named "Test - custom logs" is created
    And the "custom logs" package datasource is added to the "Test - custom logs" configuration
    And the Agent is assigned to the configuration "Test - custom logs"
    And the "Test - custom logs" configuration shows the "custom logs" datasource added
  Then there is new data in the index from agent from "custom logs" stream
