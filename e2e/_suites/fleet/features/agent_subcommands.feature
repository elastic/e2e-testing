@agent_subcommnds
Feature: Agent subcommands
  Scenarios for the Agent to test the various subcommands connecting it to Fleet.

@install
Scenario Outline: Deploying the <os> agent with install command
  When a "<os>" agent is deployed to Fleet with install command
  Then the agent is listed in Fleet as "online"
    And the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
    And system package dashboards are listed in Fleet
Examples:
| os     |
| centos |
| debian |

@stop-installed-agent
Scenario Outline: Stopping an 'installed' <os> agent stops backend processes
  Given a "<os>" agent is deployed to Fleet with install command
  When the "elastic-agent" process is "stopped" on the host
  Then the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host
Examples:
| os     |
| centos |
| debian |

@restart-installed-host
Scenario Outline: Restarting the installed <os> host restarts backend processes
  Given a "<os>" agent is deployed to Fleet with install command
  When the host is restarted
  Then the "elastic-agent" process is in the "started" state on the host
    And the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
Examples:
| os     |
| centos |
| debian |

@unenroll-installed-host
Scenario Outline: Un-enrolling the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with install command
  When the agent is un-enrolled
  Then the "elastic-agent" process is in the "started" state on the host
    And the agent is listed in Fleet as "inactive"
    And the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host
Examples:
| os     |
| centos |
| debian |

@reenroll-installed-host
Scenario Outline: Re-enrolling the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with install command
    And the agent is un-enrolled
    And the "elastic-agent" process is "stopped" on the host
  When the agent is re-enrolled on the host
    And the "elastic-agent" process is "started" on the host
  Then the agent is listed in Fleet as "online"
Examples:
| os     |
| centos |
| debian |

@uninstall-installed-host
Scenario Outline: Un-installing the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with install command
  When the agent is "uninstalled" on the host
  Then the "elastic-agent" process is in the "stopped" state on the host
    And the agent is listed in Fleet as "inactive"
    And the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host
    And the file system Agent folder is empty
    (Agent may end up deleted, if so we change this to 'Elastic folder does not have an Agent subfolder'

@restart-installed-host
Scenario Outline: Restarting the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with install command
  When the agent is "restarted" on the host
    And the agent is listed in Fleet as "online"
    And the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
