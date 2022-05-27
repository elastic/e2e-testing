@system_integration
Feature: System Integration
Scenarios for System Integration logs and metrics packages.

Scenario Outline: Adding <value> <integration> Integration to a Policy
  Given an agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the policy is updated to have "<integration>" set to "<value>"
  Then "<integration>" with "<value>" metrics are present in the datastreams

  @core
  Examples: System/metrics + core
    | integration    | value |
    | system/metrics | core  |

  @cpu
  Examples: System/metrics + cpu
    | integration    | value |
    | system/metrics | cpu   |

  @diskio
  Examples: System/metrics + diskio
    | integration    | value  |
    | system/metrics | diskio |

  @fsstat
  Examples: System/metrics + fsstat
    | integration    | value  |
    | system/metrics | fsstat |

  @load
  Examples: System/metrics + load
    | integration    | value |
    | system/metrics | load  |

  @memory
  Examples: System/metrics + memory
    | integration    | value  |
    | system/metrics | memory |

  @network
  Examples: System/metrics + network
    | integration    | value   |
    | system/metrics | network |

  @process
  Examples: System/metrics + process
    | integration    | value   |
    | system/metrics | process |

  @socket_summary
  Examples: System/metrics + socket_summary
    | integration    | value          |
    | system/metrics | socket_summary |

  @uptime
  Examples: System/metrics + uptime
    | integration    | value  |
    | system/metrics | uptime |

  @process_summary
  Examples: System/metrics + process.summary
    | integration    | value           |
    | system/metrics | process.summary |

  @filesystem
  Examples: System/metrics + filesystem
    | integration    | value      |
    | system/metrics | filesystem |

  @deploy-logfile-integration-with-syslog
  Examples: logfile + syslog
    | integration | value  |
    | logfile     | syslog |
