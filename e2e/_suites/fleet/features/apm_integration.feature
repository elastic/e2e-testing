@apm_server
Feature: APM Integration
Scenarios for APM

@install
Scenario Outline: Deploying a <image> stand-alone agent with the Elastic APM integration
  Given a "<image>" stand-alone agent is deployed
    And the stand-alone agent is listed in Fleet as "online"
  When the "Elastic APM" integration is "added" in the policy
  Then the "apm-server" process is in the "started" state on the host

@default
Examples: default
  | image   |
  | default |

@ubi8
Examples: Ubi8
| image   |
| ubi8    |
