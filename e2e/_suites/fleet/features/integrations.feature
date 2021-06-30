@integrations
Feature: Integrations
  Scenarios for operating integrations

@install
Scenario Outline: Adding an Integration to a Policy
  When the "<integration>" integration is "added" in the policy
  Then the "<integration>" datasource is shown in the policy as added
Examples:
  | integration |
  | Elastic APM |
  | Endpoint    |
  | Linux       |
