@upgrade_agent
Feature: Upgrade Agent
  Scenarios for upgrading the Agent from past releases.

Scenario Outline: Upgrading an installed agent from <stale-version>
  Given a "<stale-version>" stale agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
    And certs are installed
    And the "elastic-agent" process is "restarted" on the host
  When agent is upgraded to "latest" version
  Then agent is in "latest" version
Examples: Stale versions
| stale-version |
| latest |
| 8.2.0 |
| 8.1.2 |
| 8.1.1 |
| 8.1.0 |
| 7.17-SNAPSHOT |
