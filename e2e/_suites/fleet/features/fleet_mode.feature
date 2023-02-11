@fleet_mode
Feature: Fleet Mode
  Scenarios for the Agent in Fleet mode connecting to Fleet application.

Background: Setting up kibana instance with the default profile
  Given kibana uses "default" profile

@install
Scenario Outline: Deploying the agent
  Given an agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is in the "started" state on the host
  Then the agent is listed in Fleet as "online"
    And system package dashboards are listed in Fleet

@install-including-tags
Scenario Outline: Deploying the agent including command line --tag for tags
  When an agent is deployed to Fleet with "tar" installer and "--tag=production,linux" flags
  Then the agent is listed in Fleet as "online"
    And the elastic agent index contains the tags

# @enroll
# Scenario Outline: Deploying the agent with enroll and then run on rpm and deb
#   Given an agent is deployed to Fleet
#   When the "elastic-agent" process is in the "started" state on the host
#   Then the agent is listed in Fleet as "online"
#     And system package dashboards are listed in Fleet

# @centos
# Examples: Centos
# | os     |
# | centos |

# @debian
# Examples: Debian
# | os     |
# | debian |

@restart-agent
Scenario Outline: Restarting the installed agent
  Given an agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "restarted" on the host
  Then the agent is listed in Fleet as "online"

@unenroll
Scenario Outline: Un-enrolling the agent deactivates the agent
  Given an agent is deployed to Fleet with "tar" installer
  When the agent is un-enrolled
  Then the agent is listed in Fleet as "inactive"

@reenroll
Scenario Outline: Re-enrolling the agent activates the agent in Fleet
  Given an agent is deployed to Fleet with "tar" installer
    And the agent is un-enrolled
    And the "elastic-agent" process is "stopped" on the host
    And the agent is re-enrolled on the host
  When the "elastic-agent" process is "started" on the host
  Then the agent is listed in Fleet as "online"

@revoke-token
Scenario Outline: Revoking the enrollment token for the agent
  Given an agent is deployed to Fleet with "tar" installer
  When the enrollment token is revoked
  Then an attempt to enroll a new agent fails

@uninstall-host
Scenario Outline: Un-installing the installed agent
  Given an agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "uninstalled" on the host
  Then the file system Agent folder is empty
    And the agent is listed in Fleet as "offline"

