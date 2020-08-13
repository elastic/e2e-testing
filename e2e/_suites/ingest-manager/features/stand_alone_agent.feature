@stand_alone_mode
Feature: Stand-alone Agent
  Scenarios for a standalone mode Elastic Agent in Ingest Manager, where an Elasticseach
  and a Kibana instances are already provisioned, so that the Agent is able to communicate
  with them

@start-agent
Scenario: Starting the agent starts backend processes
  When a stand-alone agent is deployed
  Then the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host

@deploy-stand-alone
Scenario: Deploying a stand-alone agent
  When a stand-alone agent is deployed
  Then there is new data in the index from agent

@stop-agent
Scenario: Stopping the agent container stops data going into ES
  Given a stand-alone agent is deployed
  When the "elastic-agent" docker container is stopped
  Then there is no new data in the index after agent shuts down
