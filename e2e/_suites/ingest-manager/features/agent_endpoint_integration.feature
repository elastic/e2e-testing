@agent_endpoint_integration
Feature: Agent Endpoint Integration
  Scenarios for Agent to deploy Endpoint and sending data to Ingest Manager and Elasticsearch.

@deploy-endpoint-with-agent
Scenario: Adding the Endpoint Integration to an Agent makes the host to show in Security App
  Given a "centos" agent is deployed to Fleet
  When the "latest" version of the "Endpoint" package is installed
    And the "Endpoint" integration is "added" in the "default" configuration
  Then the "Endpoint" datasource is shown in the "default" configuration as added
    And the host name is shown in the Security App as "online"
    
@endpoint-policy-check
Scenario: Deploying an Endpoint makes policies to appear in the Security App
  When an Endpoint is successfully deployed with a "centos" Agent
  Then the policy response will be shown in the Security App

@set-policy-and-check-changes
Scenario: Changing an Agent policy is reflected in the Security App
  Given an Endpoint is successfully deployed with a "centos" Agent
  When the policy is updated to have malware in detect mode
  Then the policy will reflect the change in the Security App

@deploy-endpoint-then-unenroll-agent
Scenario: Un-enrolling Elastic Agent stops Elastic Endpoint
  Given an Endpoint is successfully deployed with a "centos" Agent
  When the agent is un-enrolled
  Then the agent is listed as "inactive" in Fleet
#    And the endpoint is not listed in Security App as online
    And the "elastic-endpoint" process is in the "stopped" state on the host

@deploy-endpoint-then-remove-it-from-configuration
Scenario: Removing Endpoint from Agent configuration stops the connected Endpoint
  Given an Endpoint is successfully deployed with a "centos" Agent
  When the "Endpoint" integration is "removed" in the "default" configuration
  Then the agent is listed as "inactive" in Fleet
#    And the endpoint is not listed in Security App as online
    And the "elastic-endpoint" process is in the "stopped" state on the host
