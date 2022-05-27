@linux_integration
Feature: Linux Integration
  Scenarios for Linux integration

Scenario Outline: Adding the Linux Integration to an Agent ...
  Given an agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the "Linux" integration is "added" in the policy
  Then a Linux data stream exists with some data
