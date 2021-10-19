@policies
Feature: Preconfigured Policies
  Scenarios for Preconfigured Policies

Scenario Outline: Enrolling an agent in a preconfigured policy
  Given kibana uses "preconfigured-policies" profile
  And agent uses "Test preconfigured policy" policy
  And a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is in the "started" state on the host
  Then the agent is listed in Fleet as "online"
  And the agent run the "Test preconfigured policy" policy
  

@debian
Examples: Debian
| os     |
| debian |
