@apm_server
Feature: APM Integration
Scenarios for APM

@install
Scenario Outline: Deploying a <image> stand-alone agent
  Given a "<image>" stand-alone agent is deployed
    And the stand-alone agent is listed in Fleet as "online"
  When the "Elastic APM" integration is "added" in the policy
  Then the "Elastic APM" datasource is shown in the policy as added
    And the "apm-server" process is in the "started" state on the host

@default
Examples: default
  | image   |
  | default |

@cloud
Scenario Outline: Deploying a <image> stand-alone agent on cloud
  When a "<image>" stand-alone agent is deployed on cloud
  Then the "apm-server" process is in the "started" state on the host

@default
Examples: default
  | image   |
  | default |
