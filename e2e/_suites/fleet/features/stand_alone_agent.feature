@stand_alone_agent
Feature: Stand-alone Agent
  Scenarios for a standalone mode Elastic Agent in Fleet, where an Elasticseach
  and a Kibana instances are already provisioned, so that the Agent is able to communicate
  with them

@deploy-stand-alone
Scenario Outline: Deploying a <image> stand-alone agent
  When a stand-alone agent is deployed with a "<image>" image
  Then there is new data in the index from agent

@default
Examples: default
| image   |
| default |

@ubi8
Examples: Ubi8
| image   |
| ubi8    |

@stop-agent
Scenario Outline: Stopping the <image> agent container stops data going into ES
  Given a stand-alone agent is deployed with a "<image>" image
  When the "elastic-agent" docker container is stopped
  Then there is no new data in the index after agent shuts down

@default
Examples: default
| image   |
| default |

@ubi8
Examples: Ubi8
| image   |
| ubi8    |

@run_fleet_server
Scenario Outline: Deploying a <image> stand-alone agent with fleet server mode
  When a stand-alone agent is deployed with a "<image>" image and fleet server mode
  Then the agent is listed in Fleet as "online"

@default
Examples: default
  | image   |
  | default |

@ubi8
Examples: Ubi8
  | image  |
  | ubi8   |
