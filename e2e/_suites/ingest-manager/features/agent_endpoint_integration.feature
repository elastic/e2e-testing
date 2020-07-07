@agent_endpoint_integration
Feature: Fleet Mode Agent with Endpoint Integration
  Scenarios for Agent to deploy Endpoint and sending data to Ingest Manager and Elasticsearch.

@deploy-endpoint-with-agent
Scenario: Deploy Endpoint with Agent and see host in Security App
  Given an agent is deployed to Fleet
  When the "Endpoint" latest package version is installed successfully
    And a "Endpoint" package datasource is added to the 'default' configuration
  Then the "default" configuration shows the "Endpoint" datasource added
    And the host name is shown in the security app hosts list api
    
@endpoint-policy-check
Scenario: Deploy Endpoint and see policy return successful in Security App
  Given an Endpoint is successfully deployed with Agent
  Then the policy response will be listed in the security app api

@set-policy-and-check-changes
Scenario: Deploy Endpoint and change default policy and verify in Security App
  Given an Endpoint is successfully deployed with Agent
  When the policy is updated to have malware in detect mode
  Then the policy response with detect mode settings will be listed in the security app api

@deploy-endpoint-then-unenroll-agent-and-verify
Scenario: Deploy Endpoint and then un-enroll Agent to confirm Endpoint shuts down
  Given an Endpoint is successfully deployed with Agent
  When the agent is un-enrolled
  Then the agent is not listed as online in Fleet
    And the endpoint is not listed as on-line in Security App
    And the endpoint process is stopped on the host
