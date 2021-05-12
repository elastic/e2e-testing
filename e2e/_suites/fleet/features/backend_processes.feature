@backend_processes
Feature: Backend Processes
  Scenarios for the Elastic Agent verifying backend processes are started and stopped after elastic-agent.

@install
Scenario Outline: Deploying the <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is in the "started" state on the host
  Then there are "2" instances of the "filebeat" process in the "started" state
    And there are "2" instances of the "metricbeat" process in the "started" state

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@enroll
Scenario Outline: Deploying the <os> agent with enroll and then run on rpm and deb
  Given a "<os>" agent is deployed to Fleet with "systemd" installer
  When the "elastic-agent" process is in the "started" state on the host
  Then there are "2" instances of the "filebeat" process in the "started" state
    And there are "2" instances of the "metricbeat" process in the "started" state

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@stop-agent
Scenario Outline: Stopping the <os> agent stops backend processes
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "stopped" on the host
  Then the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@restart-agent
Scenario Outline: Restarting the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "restarted" on the host
  Then there are "2" instances of the "filebeat" process in the "started" state
    And there are "2" instances of the "metricbeat" process in the "started" state

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@restart-host
Scenario Outline: Restarting the <os> host with persistent agent restarts backend processes
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the host is restarted
  Then the "elastic-agent" process is in the "started" state on the host
    And there are "2" instances of the "filebeat" process in the "started" state
    And there are "2" instances of the "metricbeat" process in the "started" state

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@unenroll
Scenario Outline: Un-enrolling the <os> agent stops backend processes
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the agent is un-enrolled
  Then the "elastic-agent" process is in the "started" state on the host
    And the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@reenroll
Scenario Outline: Re-enrolling the <os> agent starts the elastic-agent process
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is un-enrolled
    And the "elastic-agent" process is "stopped" on the host
  When the agent is re-enrolled on the host
  Then the "elastic-agent" process is "started" on the host

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@uninstall-host
Scenario Outline: Un-installing the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "uninstalled" on the host
  Then the "elastic-agent" process is in the "stopped" state on the host
    And the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@deploy-endpoint-then-unenroll-agent
Scenario Outline: Un-enrolling Elastic Agent stops Elastic Endpoint
  Given an "Endpoint" is successfully deployed with a "<os>" Agent using "tar" installer
  When the agent is un-enrolled
  Then the "elastic-endpoint" process is in the "stopped" state on the host

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@deploy-endpoint-then-remove-it-from-policy
Scenario Outline: Removing Endpoint from Agent policy stops the connected Endpoint
  Given an "Endpoint" is successfully deployed with a "<os>" Agent using "tar" installer
  When the "Endpoint Security" integration is "removed" in the policy
  Then the "elastic-endpoint" process is in the "stopped" state on the host

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |
