@linux_integration
Feature: Linux Integration
  Scenarios for Linux integration

@ingest
Scenario Outline: Adding the Linux Integration to an Agent ...
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the "Linux" integration is "added" in the policy
  Then the "Linux" datasource is shown in the policy as added
    And a Linux data stream exists with some data

@centos
Examples: Centos
  | os     |
  | centos |

@debian
Examples: Debian
  | os     |
  | debian |
