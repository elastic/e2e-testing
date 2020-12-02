@fleet_mode_agent
Feature: Fleet Mode Agent
  Scenarios for the Agent in Fleet mode connecting to Fleet application.

@install
Scenario Outline: Deploying the <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is in the "started" state on the host
  Then the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
    And the agent is listed in Fleet as "online"
    And system package dashboards are listed in Fleet
Examples:
| os     |
| centos |
| debian |

@enroll
Scenario Outline: Deploying the <os> agent with enroll and then run on rpm and deb
  Given a "<os>" agent is deployed to Fleet with "systemd" installer
  When the "elastic-agent" process is in the "started" state on the host
  Then the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
    And the agent is listed in Fleet as "online"
    And system package dashboards are listed in Fleet
Examples:
| os     |
| centos |
| debian |

@stop-agent
Scenario Outline: Stopping the <os> agent stops backend processes
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "stopped" on the host
  Then the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host
Examples:
| os     |
| centos |
| debian |

@upgrade-agent
Scenario Outline: Upgrading the installed <os> agent
  Given a "<os>" agent "stale" is deployed to Fleet with "<installer>" installer
  When agent is upgraded to version "latest"
  Then wait for "2m"
  Then agent is in version "latest"
    And the agent is listed in Fleet as "online"
Examples:
| os     | installer |
| debian | tar       |  

@restart-agent
Scenario Outline: Restarting the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "restarted" on the host
  Then the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
    And the agent is listed in Fleet as "online"
Examples:
| os     |
| centos |
| debian |

@restart-host
Scenario Outline: Restarting the <os> host with persistent agent restarts backend processes
  Given a "<os>" agent is deployed to Fleet with "tar" installer
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
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the agent is un-enrolled
  Then the "elastic-agent" process is in the "started" state on the host
    And the agent is listed in Fleet as "inactive"
    And the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host
Examples:
| os     |
| centos |
| debian |

@reenroll
Scenario Outline: Re-enrolling the <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is un-enrolled
    And the "elastic-agent" process is "stopped" on the host
  When the agent is re-enrolled on the host
  Then the "elastic-agent" process is "started" on the host
    And the agent is listed in Fleet as "online"
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
  Then the "elastic-agent" process is in the "stopped" state on the host
    And the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host
    And the file system Agent folder is empty
    And the agent is listed in Fleet as "offline"
Examples:
| os     |
| centos |
| debian |
