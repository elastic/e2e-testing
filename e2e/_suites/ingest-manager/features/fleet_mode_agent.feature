@fleet_mode
Feature: Fleet Mode Agent
  Scenarios for the Agent in Fleet mode connecting to Ingest Manager application.

@enroll
Scenario: Deploying an agent
  When a "centos" agent is deployed to Fleet
  Then the agent is listed in Fleet as online
    And system package dashboards are listed in Fleet

@start-agent
Scenario: Starting the agent starts backend processes
  When a "centos" agent is deployed to Fleet
  Then the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host

@stop-agent
Scenario: Stopping the agent stops backend processes
  Given a "centos" agent is deployed to Fleet
  When the "elastic-agent" process is "stopped" on the host
  Then the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host

@restart-host
Scenario Outline: Restarting the <os> host with persistent agent restarts backend processes
  Given a "<os>" agent is deployed to Fleet
  When the host is restarted
  Then the "elastic-agent" process is in the "started" state on the host
    And the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
Examples:
| os         |
| centos     |
| debian     |
| redhat-ubi |

@unenroll
Scenario: Un-enrolling an agent
  Given a "centos" agent is deployed to Fleet
  When the agent is un-enrolled
  Then the "elastic-agent" process is in the "started" state on the host
    But the agent is not listed as online in Fleet

@reenroll
Scenario: Re-enrolling an agent
  Given a "centos" agent is deployed to Fleet
    And the agent is un-enrolled
    And the "elastic-agent" process is "stopped" on the host
  When the agent is re-enrolled on the host
    And the "elastic-agent" process is "started" on the host
  Then the agent is listed in Fleet as online

@revoke-token
Scenario: Revoking the enrollment token for an agent
  Given a "centos" agent is deployed to Fleet
  When the enrollment token is revoked
  Then an attempt to enroll a new agent fails
