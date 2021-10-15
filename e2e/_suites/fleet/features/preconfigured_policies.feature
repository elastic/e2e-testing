@policies
Feature: Preconfigured Policies
  Scenarios for Preconfigured Policies

Scenario Outline: Example using Kibana with custom policy
  Given kibana uses "preconfigured-policies" profile
    And a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is in the "started" state on the host
  Then the agent is listed in Fleet as "online"
    And system package dashboards are listed in Fleet

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

Scenario Outline: Example using Kibana with default config
  Given kibana uses "default" profile
    And a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is in the "started" state on the host
  Then the agent is listed in Fleet as "online"
    And system package dashboards are listed in Fleet

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |
