@agent_endpoint_integration
Feature: Agent Endpoint Integration
  Scenarios for Agent to deploy Endpoint and sending data to Ingest Manager and Elasticsearch.

@deploy-endpoint-with-agent
Scenario: Adding the Endpoint Integration to an Agent makes the host to show in Security App
  Given an agent is deployed to Fleet
  When the "Endpoint" latest package version is installed successfully
    And the "Endpoint" integration is "added" in the "default" configuration
  Then the "default" configuration shows the "Endpoint" datasource added
    And the host name is shown in the Security App
    
@endpoint-policy-check
Scenario: Deploying an Endpoint makes policies to appear in the Security App
  Given an Endpoint is successfully deployed with Agent
  Then the policy response will be shown in the Security App

@set-policy-and-check-changes
Scenario: Changing an Agent policy is reflected in the Security App
  Given an Endpoint is successfully deployed with Agent
  When the policy is updated to have malware in detect mode
  Then the policy will reflect the change in the Security App

@deploy-endpoint-then-unenroll-agent
Scenario: Un-enrolling Elastic Agent stops Elastic Endpoint
  Given an Endpoint is successfully deployed with Agent
  When the agent is un-enrolled
  Then the agent is not listed in Fleet as online
    And the endpoint is not listed in Security App as online
    And the "elastic-endpoint" process is "stopped" on the host

@deploy-endpoint-then-remove-it-from-configuration
Scenario: Removing Endpoint from Agent configuration stops the connected Endpoint
  Given an Endpoint is successfully deployed with Agent
  When the "Endpoint" integration is "removed" in the "default" configuration
  Then the agent is listed in Fleet as online
    And the endpoint is not listed in Security App as online
    And the "elastic-endpoint" process is "stopped" on the host
