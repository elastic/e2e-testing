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
<<<<<<< HEAD
| 7.16.0 |
| 7.15.0 |
| 7.14.0 |
=======
| 8.4.0 |
| 8.3.0 |
| 8.2.0 |
| 8.1.3 |
| 8.1.0 |
| 7.17.8 |
>>>>>>> d5541388 (fix: remove Helm Chart tests (#3285))
