@agent_endpoint_integration
Feature: Agent Endpoint Integration
  Scenarios for Agent to deploy Endpoint and sending data to Ingest Manager and Elasticsearch.

@deploy-endpoint-with-agent
Scenario: Adding the Endpoint Integration to an Agent makes the host to show in Security App
  Given an agent is deployed to Fleet
  When the "Endpoint" latest package version is installed successfully
    And the "Endpoint" integration is "added" in the "default" configuration
  Then the "default" configuration shows the "Endpoint" datasource added
    And the host name is shown in the security app
    
@endpoint-policy-check
Scenario: Deploy Endpoint and see policy return successful in Security App
  Given an Endpoint is successfully deployed with Agent
  Then the policy response will be listed in the security app

@set-policy-and-check-changes
Scenario: Deploy Endpoint and change default policy and verify in Security App
  Given an Endpoint is successfully deployed with Agent
  When the policy is updated to have malware in detect mode
  Then the policy response will be listed in the security app

@deploy-endpoint-then-unenroll-agent-and-verify
Scenario: Deploy Endpoint and then un-enroll Agent to confirm Endpoint stops
  Given an Endpoint is successfully deployed with Agent
  When the agent is un-enrolled
  Then the agent is not listed in Fleet as online
    And the endpoint is not listed in Security App as online
    And the "elastic-endpoint" process is "stopped" on the host

@deploy-endpoint-then-remove-it-from-configuration-and-verify
Scenario: Deploy Endpoint and remove it from the configuration then confirm Endpoint stops
  Given an Endpoint is successfully deployed with Agent
  When the "Endpoint" integration is "removed" in the "default" configuration
  Then the agent is listed in Fleet as online
    And the endpoint is not listed in Security App as online
    And the "elastic-endpoint" process is "stopped" on the host
