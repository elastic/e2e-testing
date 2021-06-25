@deploy-system_integration
Feature: System Integration
Scenarios for System Integration logs and metrics packages.

@deploy-system_integration-with-core
Scenario Outline: Adding core System Integration to an Policy
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "core"
    And we verify that "system/metrics" with "core" metrics in the datastreams

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
 | debian |

@deploy-system_integration-with-cpu
Scenario Outline: Adding cpu System Integration to an Policy
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "cpu"
    And we verify that "system/metrics" with "cpu" metrics in the datastreams

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@deploy-system_integration-with-diskio
Scenario Outline: Adding diskio System Integration to an Policy
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "diskio"
    And we verify that "system/metrics" with "diskio" metrics in the datastreams

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@deploy-system_integration-with-fsstat
Scenario Outline: Adding fsstat System Integration to an Policy
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "fsstat"
    And we verify that "system/metrics" with "fsstat" metrics in the datastreams

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@deploy-system_integration-with-load
Scenario Outline: Adding load System Integration to an Policy
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "load"
    And we verify that "system/metrics" with "load" metrics in the datastreams

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@deploy-system_integration-with-memory
Scenario Outline: Adding memory System Integration to an Policy
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "memory"
    And we verify that "system/metrics" with "memory" metrics in the datastreams

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@deploy-system_integration-with-network
Scenario Outline: Adding network System Integration to an Policy
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "network"
    And we verify that "system/metrics" with "network" metrics in the datastreams

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@deploy-system_integration-with-process
Scenario Outline: Adding process System Integration to an Policy
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "process"
    And we verify that "system/metrics" with "process" metrics in the datastreams

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@deploy-system_integration-with-socket_summary
Scenario Outline: Adding socket_summary System Integration to an Policy
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "socket_summary"
    And we verify that "system/metrics" with "socket_summary" metrics in the datastreams

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@deploy-system_integration-with-uptime
Scenario Outline: Adding uptime System Integration to an Policy
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "uptime"
    And we verify that "system/metrics" with "uptime" metrics in the datastreams

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

#  @deploy-system_integration-with-process_summary
#  Scenario Outline: Adding process_summary System Integration to an Policy
#    Given a "<os>" agent is deployed to Fleet with "tar" installer
#    And the agent is listed in Fleet as "online"
#    When the policy is updated to have "system/metrics" set to "process_summary"
#    And we verify that "system/metrics" with "process_summary" metrics in the datastreams
#
#    @centos
#    Examples: Centos
#      | os     |
#      | centos |
#
#    @debian
#    Examples: Debian
#      | os     |
#      | debian |

#  @deploy-system_integration-with-filesystem
#  Scenario Outline: Adding the System Integration to an Policy
#    Given a "<os>" agent is deployed to Fleet with "tar" installer
#    And the agent is listed in Fleet as "online"
#    When the policy is updated to have "system/metrics" set to "filesystem"
#    And we verify that "system/metrics" with "filesystem" metrics in the datastreams
#
#    @centos
#    Examples: Centos
#      | os     |
#      | centos |

#    @debian
#    Examples: Debian
#      | os     |
#      | debian |

#  @deploy-logfile-for-system-auth
#  Scenario Outline: Adding the System Integration to an Policy
#    Given a "<os>" agent is deployed to Fleet with "tar" installer
#    And the agent is listed in Fleet as "online"
#    When the policy is updated to have "logfile" set to "syslog"
#    And we verify that "logfile" with "syslog" metrics in the datastreams
#
#    @centos
#    Examples: Centos
#      | os     |
#      | centos |

#    @debian
#    Examples: Debian
#      | os     |
#      | debian |
