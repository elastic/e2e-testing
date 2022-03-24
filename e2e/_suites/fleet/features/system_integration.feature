@system_integration
Feature: System Integration
Scenarios for System Integration logs and metrics packages.

Scenario Outline: Adding <value> System Integration to an Policy
  Given an agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "<value>"
  Then "system/metrics" with "<value>" metrics are present in the datastreams

  @deploy-system_integration-with-core
  Examples: core
    | value |
    | core  |

  @deploy-system_integration-with-cpu
  Examples: cpu
    | value |
    | cpu  |

  @deploy-system_integration-with-diskio
  Examples: diskio
    | value   |
    | diskio  |

  @deploy-system_integration-with-fsstat
  Examples: fsstat
    | value   |
    | fsstat  |

  @deploy-system_integration-with-load
  Examples: load
    | value |
    | load  |

  @deploy-system_integration-with-memory
  Examples: memory
    | value  |
    | memory |

  @deploy-system_integration-with-network
  Examples: network
    | value   |
    | network |

  @deploy-system_integration-with-process
  Examples: process
    | value   |
    | process |

  @deploy-system_integration-with-socket_summary
  Examples: socket_summary
    | value          |
    | socket_summary |

  @deploy-system_integration-with-uptime
  Examples: uptime
    | value  |
    | uptime |

  @deploy-system_integration-with-process_summary
  Examples: process.summary
    | value           |
    | process.summary |

  @deploy-system_integration-with-filesystem
  Examples: filesystem
    | value      |
    | filesystem |

#  @deploy-logfile-for-system-auth
#  Scenario Outline: Adding the System Integration to an Policy
#    Given a "<os>" agent is deployed to Fleet with "tar" installer
#    And the agent is listed in Fleet as "online"
#    When the policy is updated to have "logfile" set to "syslog"
#    And verify that "logfile" with "syslog" metrics in the datastreams
#
#    @centos
#    Examples: Centos
#      | os     |
#      | centos |

#    @debian
#    Examples: Debian
#      | os     |
#      | debian |
