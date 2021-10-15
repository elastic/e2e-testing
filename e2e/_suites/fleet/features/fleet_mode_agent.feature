@fleet_mode_agent
Feature: Fleet Mode Agent
  Scenarios for the Agent in Fleet mode connecting to Fleet application.

Background: Setting up kibana instance with the default profile
  Given kibana uses "default" profile

@install
Scenario Outline: Deploying the <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is in the "started" state on the host
  Then the agent is listed in Fleet as "online"
    And system package dashboards are listed in Fleet

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@enroll
@skip:windows
Scenario Outline: Deploying the <os> agent with enroll and then run on rpm and deb
  Given a "<os>" agent is deployed to Fleet
  When the "elastic-agent" process is in the "started" state on the host
  Then the agent is listed in Fleet as "online"
    And system package dashboards are listed in Fleet

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@restart-agent
@skip:windows
Scenario Outline: Restarting the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "restarted" on the host
  Then the agent is listed in Fleet as "online"

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@unenroll
@skip:windows
Scenario Outline: Un-enrolling the <os> agent deactivates the agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the agent is un-enrolled
  Then the agent is listed in Fleet as "inactive"

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@reenroll
@skip:windows
Scenario Outline: Re-enrolling the <os> agent activates the agent in Fleet
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is un-enrolled
    And the "elastic-agent" process is "stopped" on the host
    And the agent is re-enrolled on the host
  When the "elastic-agent" process is "started" on the host
  Then the agent is listed in Fleet as "online"

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@revoke-token
Scenario Outline: Revoking the enrollment token for the <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the enrollment token is revoked
  Then an attempt to enroll a new agent fails

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

@uninstall-host
@skip:windows
Scenario Outline: Un-installing the installed <os> agent
  Given a "<os>" agent is deployed to Fleet with "tar" installer
  When the "elastic-agent" process is "uninstalled" on the host
  Then the file system Agent folder is empty

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |
