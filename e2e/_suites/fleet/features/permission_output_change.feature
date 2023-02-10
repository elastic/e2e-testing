@permission_change
Feature: Permission output change
Scenarios for Permission Change

@add-linux-integration
Scenario Outline: Adding the Linux Integration to an Agent changing Default API key
  Given an agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the "Linux" integration is "added" in the policy
  Then a Linux data stream exists with some data
    And the output permissions has "changed"

@update-system-metrics-integration
Scenario Outline: Updating the Integration on an Agent not changing Default API key
  Given an agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "core"
  Then the output permissions has "not changed"
