@stand_alone_mode
Feature: Stand-alone Mode
  Scenarios for a the Elastic Agent running in stand-alone mode

@start-agent
Scenario Outline: Starting the <image> agent starts backend processes
  When a "<image>" stand-alone agent is deployed
  Then there are "2" instances of the "filebeat" process in the "started" state
    And there are "2" instances of the "metricbeat" process in the "started" state

@default
Examples: default
| image   |
| default |

@ubi8
@skip
Examples: Ubi8
| image   |
| ubi8    |

@deploy-stand-alone
Scenario Outline: Deploying a <image> stand-alone agent
  When a "<image>" stand-alone agent is deployed
  Then there is new data in the index from agent

@default
Examples: default
| image   |
| default |

@ubi8
@skip
Examples: Ubi8
| image   |
| ubi8    |

@stop-agent
Scenario Outline: Stopping the <image> agent container stops data going into ES
  Given a "<image>" stand-alone agent is deployed
  When the "elastic-agent" docker container is stopped
  Then there is no new data in the index after agent shuts down

@default
Examples: default
| image   |
| default |

@ubi8
@skip
Examples: Ubi8
| image   |
| ubi8    |

@bootstrap-fleet-server
@skip
Scenario Outline: Bootstrapping Fleet Server from a <image> stand-alone Elastic Agent
  When a "<image>" stand-alone agent is deployed with fleet server mode
  Then the stand-alone agent is listed in Fleet as "online"
    And there are "1" instances of the "fleet-server" process in the "started" state

@default
Examples: default
  | image   |
  | default |

@ubi8
@skip
Examples: Ubi8
  | image   |
  | ubi8    |

@start-stand-alone-agent-with-process_summary
Scenario Outline: Adding the process_summary System Integration to an stand-alone-agent
  Given a "<image>" stand-alone agent is deployed
    And the stand-alone agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "process.summary"
  Then "system/metrics" with "process.summary" metrics are present in the datastreams

@ubi8
@skip
Examples: Ubi8
| image   |
| ubi8 |

@default
Examples: default
| image     |
| default |

@start-stand-alone-agent-with-core
Scenario Outline: Adding the core System Integration to an stand-alone-agent
  Given a "<image>" stand-alone agent is deployed
    And the stand-alone agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "core"
  Then "system/metrics" with "core" metrics are present in the datastreams

@ubi8
@skip
Examples: Ubi8
| image |
| ubi8  |

@default
Examples: default
| image   |
| default |

@start-stand-alone-agent-with-cpu
Scenario Outline: Adding the cpu System Integration to an stand-alone-agent
  Given a "<image>" stand-alone agent is deployed
    And the stand-alone agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "cpu"
  Then "system/metrics" with "cpu" metrics are present in the datastreams

@ubi8
@skip
Examples: Ubi8
| image |
| ubi8  |

@default
Examples: default
| image   |
| default |

@start-stand-alone-agent-with-diskio
Scenario Outline: Adding the diskio System Integration to an stand-alone-agent
  Given a "<image>" stand-alone agent is deployed
    And the stand-alone agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "diskio"
  Then "system/metrics" with "diskio" metrics are present in the datastreams

@ubi8
@skip
Examples: Ubi8
| image |
| ubi8  |

@default
Examples: default
| image   |
| default |

@start-stand-alone-agent-with-fsstat
Scenario Outline: Adding the fsstat System Integration to an stand-alone-agent
  Given a "<image>" stand-alone agent is deployed
    And the stand-alone agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "fsstat"
  Then "system/metrics" with "fsstat" metrics are present in the datastreams

@ubi8
@skip
Examples: Ubi8
| image |
| ubi8  |

@default
Examples: default
| image   |
| default |

@start-stand-alone-agent-with-load
Scenario Outline: Adding the load System Integration to an stand-alone-agent
  Given a "<image>" stand-alone agent is deployed
    And the stand-alone agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "load"
  Then "system/metrics" with "load" metrics are present in the datastreams

@ubi8
@skip
Examples: Ubi8
| image |
| ubi8  |

@default
Examples: default
| image   |
| default |

@start-stand-alone-agent-with-memory
Scenario Outline: SAdding the memory System Integration to an stand-alone-agent
  Given a "<image>" stand-alone agent is deployed
    And the stand-alone agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "memory"
  Then "system/metrics" with "memory" metrics are present in the datastreams

@ubi8
@skip
Examples: Ubi8
| image |
| ubi8  |

@default
Examples: default
| image   |
| default |

@start-stand-alone-agent-with-network
Scenario Outline: Adding the network System Integration to an stand-alone-agent
  Given a "<image>" stand-alone agent is deployed
    And the stand-alone agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "network"
  Then "system/metrics" with "network" metrics are present in the datastreams

@ubi8
@skip
Examples: Ubi8
| image |
| ubi8  |

@default
Examples: default
| image   |
| default |

@start-stand-alone-agent-with-process
Scenario Outline: Adding the process System Integration to an stand-alone-agent
  Given a "<image>" stand-alone agent is deployed
    And the stand-alone agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "process"
  Then "system/metrics" with "process" metrics are present in the datastreams

@ubi8
@skip
Examples: Ubi8
| image |
| ubi8  |

@default
Examples: default
| image   |
| default |

@start-stand-alone-agent-with-socket_summary
Scenario Outline: Adding the socket_summary System Integration to an stand-alone-agent
  Given a "<image>" stand-alone agent is deployed
    And the stand-alone agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "socket_summary"
  Then "system/metrics" with "socket_summary" metrics are present in the datastreams

@ubi8
@skip
Examples: Ubi8
| image |
| ubi8  |

@default
Examples: default
| image   |
| default |

@start-stand-alone-agent-with-uptime
Scenario Outline: Adding the uptime System Integration to an stand-alone-agent
  Given a "<image>" stand-alone agent is deployed
    And the stand-alone agent is listed in Fleet as "online"
  When the policy is updated to have "system/metrics" set to "uptime"
  Then "system/metrics" with "uptime" metrics are present in the datastreams

@ubi8
@skip
Examples: Ubi8
| image |
| ubi8  |

@default
Examples: default
  | image   |
  | default |
