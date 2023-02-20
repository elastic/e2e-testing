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
      | stale-version    |
      | latest           |
      | 7.17.10-SNAPSHOT |

# These are the version of elastic agent that still have the bug solved in
# https://github.com/elastic/elastic-agent/pull/1791
    @skip
    Examples: Skipped stale versions
      | stale-version |
      | 7.17.8        |
      | 7.16.0        |
      | 7.15.0        |
      | 7.14.0        |
