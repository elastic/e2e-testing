@agent_endpoint_integration @skip:arm64
Feature: Agent Endpoint Integration
  Scenarios for Agent to deploy Endpoint and sending data to Fleet and Elasticsearch.

@deploy-endpoint-with-agent
Scenario Outline: Adding the Endpoint Integration to an Agent makes the host to show in Security App
  Given a agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the "Endpoint Security" integration is "added" in the policy
  Then the "Endpoint Security" datasource is shown in the policy as added
    And the policy response will be shown in the Security App
    And the host name is shown in the Administration view in the Security App as "online"

@endpoint-policy-check
Scenario Outline: Deploying an Endpoint makes policies to appear in the Security App
  Given an "Endpoint" is successfully deployed with an Agent using "tar" installer
    And the agent is listed in Fleet as "online"
  When the host name is shown in the Administration view in the Security App as "online"
  Then the policy response will be shown in the Security App

@set-policy-and-check-changes
Scenario Outline: Changing an Agent policy is reflected in the Security App
  Given an "Endpoint" is successfully deployed with an Agent using "tar" installer
    And the agent is listed in Fleet as "online"
    And the host name is shown in the Administration view in the Security App as "online"
  When the policy is updated to have "malware" in "detect" mode
  Then the policy will reflect the change in the Security App

@deploy-endpoint-then-unenroll-agent
Scenario Outline: Un-enrolling Elastic Agent stops Elastic Endpoint
  Given an "Endpoint" is successfully deployed with an Agent using "tar" installer
    And the agent is listed in Fleet as "online"
    And the host name is shown in the Administration view in the Security App as "online"
  When the agent is un-enrolled
  Then the agent is listed in Fleet as "inactive"
    And the host name is not shown in the Administration view in the Security App

@deploy-endpoint-then-remove-it-from-policy
Scenario Outline: Removing Endpoint from Agent policy stops the connected Endpoint
   Given an "Endpoint" is successfully deployed with an Agent using "tar" installer
    And the agent is listed in Fleet as "online"
    And the host name is shown in the Administration view in the Security App as "online"
  When the "Endpoint Security" integration is "removed" in the policy
  Then the agent is listed in Fleet as "online"
    But the host name is not shown in the Administration view in the Security App
    And the "elastic-endpoint" process is in the "stopped" state on the host

@stop-agent-and-endpoint
Scenario Outline: Stopping the agent deployed with Endpoint stops all backend processes
   Given an "Endpoint" is successfully deployed with an Agent using "tar" installer
    And the agent is listed in Fleet as "online"
    And the host name is shown in the Administration view in the Security App as "online"
  When the "elastic-agent" process is "stopped" on the host
  Then the "elastic-endpoint" process is in the "stopped" state on the host

#@restart-host-with-endpoint-deployed
#Scenario Outline: Restarting the host with persistent agent with Endpoint restarts backend processes
#   Given an "Endpoint" is successfully deployed with an Agent using "tar" installer
#    And the agent is listed in Fleet as "online"
#    And the host name is shown in the Administration view in the Security App as "online"
#  When the host is restarted
#  Then the "elastic-agent" process is in the "started" state on the host
#    And the "elastic-endpoint" process is in the "started" state on the host

@unenroll-with-deployed-endpoint
Scenario Outline: Un-enrolling the agent with Endpoint
   Given an "Endpoint" is successfully deployed with an Agent using "tar" installer
    And the agent is listed in Fleet as "online"
    And the host name is shown in the Administration view in the Security App as "online"
  When the agent is un-enrolled
  Then the "elastic-agent" process is in the "started" state on the host
    And the agent is listed in Fleet as "inactive"
    And the "elastic-endpoint" process is in the "stopped" state on the host
