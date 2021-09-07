@permission_change
Feature: Permission output change
Scenarios for Permission Change

@adding-integration-change-permission
Scenario Outline: Adding the Linux Integration to an Agent ...
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
    And the agent get Default Api Key
  When the "Linux" integration is "added" in the policy
    And a Linux data stream exists with some data

@centos
Examples: Centos
| os     |
| centos |
