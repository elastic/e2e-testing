@fleet_mode
Feature: Fleet Mode Agent
  Scenarios for the Agent in Fleet mode connecting to Ingest Manager application.

@enroll
Scenario: Deploying an agent
  When an agent is deployed to Fleet
  Then the agent is listed in Fleet as online
    And the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
    And system package dashboards are listed in Fleet

@stop-agent
Scenario: Stopping the agent stops backend processes
  Given an agent is deployed to Fleet
  When the "elastic-agent" process is "stopped" on the host
  Then the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host

@unenroll
Scenario: Un-enrolling an agent
  Given an agent is deployed to Fleet
  When the agent is un-enrolled
  Then the "elastic-agent" process is in the "started" state on the host
    But the agent is not listed as online in Fleet

@reenroll
Scenario: Re-enrolling an agent
  Given an agent is deployed to Fleet
    And the agent is un-enrolled
    And the "elastic-agent" process is "stopped" on the host
  When the agent is re-enrolled on the host
    And the "elastic-agent" process is "started" on the host
  Then the agent is listed in Fleet as online

@revoke-token
Scenario: Revoking the enrollment token for an agent
  Given an agent is deployed to Fleet
    And the agent is un-enrolled
    And the "agent" process is "stopped" on the host
  When the enrollment token is revoked
  Then an attempt to enroll an agent with the old token fails

@package-added-to-default-config
Scenario: Execute packages API calls
  Given an agent is deployed to Fleet
    And the package list API returns successfully
  When the "Cisco" "latest" package version is installed successfully
    And the "Cisco" package datasource is added to the "default" configuration
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
