@stand_alone_mode
Feature: Stand-alone Agent
  Scenarios for a standalone mode Elastic Agent in Ingest Manager, where an Elasticseach
  and a Kibana instances are already provisioned, so that the Agent is able to communicate
  with them

@start-agent
Scenario Outline: Starting the <image> agent starts backend processes
  When a "<image>" stand-alone agent is deployed
  Then the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host
Examples:
| image   |
| default |
| ubi8    |

@deploy-stand-alone
Scenario Outline: Deploying a <image> stand-alone agent
  When a "<image>" stand-alone agent is deployed
  Then there is new data in the index from agent
Examples:
| image   |
| default |
| ubi8    |

@stop-agent
Scenario Outline: Stopping the <image> agent container stops data going into ES
  Given a "<image>" stand-alone agent is deployed
  When the "elastic-agent" docker container is stopped
  Then there is no new data in the index after agent shuts down
Examples:
| image   |
| default |
| ubi8    |
