@permission_change
Feature: Permission output change
Scenarios for Permission Change

@adding-integration-change-permission
Scenario Outline: Adding the Linux Integration to an Agent ...
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
    And the "Linux" integration is "added" in the policy
  When a Linux data stream exists with some data
  Then the default API key has "changed"

@centos
Examples: Centos
  | os     |
  | centos |

  @debian
Examples: Debian
| os     |
| debian |

@updating-integration-do-not-change-permission
Scenario Outline: Updating the Linux Integration to an Agent ...
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "core"
  Then the default API key has "not change"

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |
