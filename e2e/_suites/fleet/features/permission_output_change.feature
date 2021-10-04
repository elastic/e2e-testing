@permission_change
Feature: Permission output change
Scenarios for Permission Change

@adding-integration-change-permission
Scenario Outline: Adding the Linux Integration to an Agent ...
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
    And the agent get Default Api Key
  When the "Linux" integration is "added" in the policy
    And a Linux data stream exists with some data
  Then verify that Default Api Key is "different"

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

#@updating-integration-do-not-change-permission
#Scenario Outline: Updating the Linux Integration to an Agent ...
#  Given a "<os>" agent is deployed to Fleet with "tar" installer
#    And the agent is listed in Fleet as "online"
#    And the "Linux" integration is "added" in the policy
#  When the agent get Default Api Key
#    And the policy is updated to have "linux/metrics" set to "pageinfo"
#  Then verify that Default Api Key is "identical"
#
#@centos
#Examples: Centos
#| os     |
#| centos |