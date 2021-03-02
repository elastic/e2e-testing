@fleet_mode_agent
Feature: Fleet Mode Agent
  Scenarios for the Agent in Fleet mode connecting to Fleet application.

@install
Scenario Outline: Deploying the <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is in the "started" state on the host
  Then the agent is listed in Fleet as "online"
    And system package dashboards are listed in Fleet
Examples:
| os     |
| centos |
| debian |

@enroll
Scenario Outline: Deploying the <os> agent with enroll and then run on rpm and deb
  Given a "<os>" agent is deployed to Fleet with "systemd" installer
  When the "elastic-agent" process is in the "started" state on the host
  Then the agent is listed in Fleet as "online"
    And system package dashboards are listed in Fleet
Examples:
| os     |
| centos |
| debian |

# @upgrade-agent
@nightly
Scenario Outline: Upgrading the installed <os> agent
  Given a "<os>" agent "stale" is deployed to Fleet with "tar" installer
    And certs are installed
    And the "elastic-agent" process is "restarted" on the host
  When agent is upgraded to version "latest"
  Then agent is in version "latest"
Examples:
| os     | 
| debian | 

@restart-agent
Scenario Outline: Restarting the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "restarted" on the host
  Then the agent is listed in Fleet as "online"
Examples:
| os     |
| centos |
| debian |

@unenroll
Scenario Outline: Un-enrolling the <os> agent deactivates the agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the agent is un-enrolled
  Then the agent is listed in Fleet as "inactive"
Examples:
| os     |
| centos |
| debian |

@reenroll
Scenario Outline: Re-enrolling the <os> agent activates the agent in Fleet
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is un-enrolled
    And the "elastic-agent" process is "stopped" on the host
  When the agent is re-enrolled on the host
  Then the agent is listed in Fleet as "online"
Examples:
| os     |
| centos |
| debian |

@revoke-token
Scenario Outline: Revoking the enrollment token for the <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the enrollment token is revoked
  Then an attempt to enroll a new agent fails
Examples:
| os     |
| centos |
| debian |

@uninstall-host
Scenario Outline: Un-installing the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "uninstalled" on the host
  Then the file system Agent folder is empty
    And the agent is listed in Fleet as "offline"
Examples:
| os     |
| centos |
| debian |
