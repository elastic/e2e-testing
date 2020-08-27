@fleet_mode
Feature: Fleet Mode Agent
  Scenarios for the Agent in Fleet mode connecting to Ingest Manager application.

@enroll
Scenario Outline: Deploying the <os> agent
  When a "<os>" agent is deployed to Fleet
  Then the agent is listed in Fleet as "online"
    And system package dashboards are listed in Fleet
Examples:
| os     |
| centos |
| debian |

@start-agent
Scenario Outline: Starting the <os> agent starts backend processes
  When a "<os>" agent is deployed to Fleet
  Then the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
Examples:
| os     |
| centos |
| debian |

@stop-agent
Scenario Outline: Stopping the <os> agent stops backend processes
  Given a "<os>" agent is deployed to Fleet
  When the "elastic-agent" process is "stopped" on the host
  Then the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host
Examples:
| os     |
| centos |
| debian |

@restart-host
Scenario Outline: Restarting the <os> host with persistent agent restarts backend processes
  Given a "<os>" agent is deployed to Fleet
  When the host is restarted
  Then the "elastic-agent" process is in the "started" state on the host
    And the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
Examples:
| os     |
| centos |
| debian |

@unenroll
Scenario Outline: Un-enrolling the <os> agent
  Given a "<os>" agent is deployed to Fleet
  When the agent is un-enrolled
  Then the "elastic-agent" process is in the "started" state on the host
    And the agent is listed in Fleet as "offline"
Examples:
| os     |
| centos |
| debian |

@reenroll
Scenario Outline: Re-enrolling the <os> agent
  Given a "<os>" agent is deployed to Fleet
    And the agent is un-enrolled
    And the "elastic-agent" process is "stopped" on the host
  When the agent is re-enrolled on the host
    And the "elastic-agent" process is "started" on the host
  Then the agent is listed in Fleet as "online"
Examples:
| os     |
| centos |
| debian |

@revoke-token
Scenario Outline: Revoking the enrollment token for the <os> agent
  Given a "<os>" agent is deployed to Fleet
  When the enrollment token is revoked
  Then an attempt to enroll a new agent fails
Examples:
| os     |
| centos |
| debian |
