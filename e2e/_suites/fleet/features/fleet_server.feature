@fleet_server
Feature: Fleet Server
  Scenarios for Fleet Server, where an Elasticseach and a Kibana instances are already provisioned,
  so that the Agent is able to communicate with them using Fleet Server

@bootstrap-fleet-server
Scenario Outline: Bootstrapping Fleet Server from an <os> Elastic Agent
  When a "<os>" agent is deployed to Fleet with "tar" installer in fleet-server mode
  Then the agent is listed in Fleet as "online"

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |
