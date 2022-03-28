@system_integration
Feature: System Integration
Scenarios for System Integration logs and metrics packages.

Scenario Outline: Adding <value> <integration> Integration to a Policy
  Given an agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "<integration>" set to "<value>"
  Then "<integration>" with "<value>" metrics are present in the datastreams

  @deploy-system_integration-with-core
  Examples: System/metrics + core
    | integration    | value |
    | system/metrics | core  |

  @deploy-system_integration-with-cpu
  Examples: System/metrics + cpu
    | integration    | value |
    | system/metrics | cpu   |

  @deploy-system_integration-with-diskio
  Examples: System/metrics + diskio
    | integration    | value  |
    | system/metrics | diskio |

  @deploy-system_integration-with-fsstat
  Examples: System/metrics + fsstat
    | integration    | value  |
    | system/metrics | fsstat |

  @deploy-system_integration-with-load
  Examples: System/metrics + load
    | integration    | value |
    | system/metrics | load  |

  @deploy-system_integration-with-memory
  Examples: System/metrics + memory
    | integration    | value  |
    | system/metrics | memory |

  @deploy-system_integration-with-network
  Examples: System/metrics + network
    | integration    | value   |
    | system/metrics | network |

  @deploy-system_integration-with-process
  Examples: System/metrics + process
    | integration    | value   |
    | system/metrics | process |

  @deploy-system_integration-with-socket_summary
  Examples: System/metrics + socket_summary
    | integration    | value          |
    | system/metrics | socket_summary |

  @deploy-system_integration-with-uptime
  Examples: System/metrics + uptime
    | integration    | value  |
    | system/metrics | uptime |

  @deploy-system_integration-with-process_summary
  Examples: System/metrics + process.summary
    | integration    | value           |
    | system/metrics | process.summary |

  @deploy-system_integration-with-filesystem
  Examples: System/metrics + filesystem
    | integration    | value      |
    | system/metrics | filesystem |

  @deploy-logfile-integration-with-syslog
  Examples: logfile + syslog
    | integration | value  |
    | logfile     | syslog |
