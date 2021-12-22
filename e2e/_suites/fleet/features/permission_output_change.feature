@permission_change
Feature: Permission output change
Scenarios for Permission Change

@adding-integration-change-permission
Scenario Outline: Adding the Linux Integration to an Agent changing Default API key
  Given an agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the "Linux" integration is "added" in the policy
  Then a Linux data stream exists with some data
    And the default API key has "changed"

@updating-integration-do-not-change-permission
Scenario Outline: Updating the Integration on an Agent not changing Default API key
  Given an agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "core"
  Then the default API key has "not changed"
