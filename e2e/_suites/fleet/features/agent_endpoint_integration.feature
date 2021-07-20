@agent_endpoint_integration @skip:arm64
Feature: Agent Endpoint Integration
  Scenarios for Agent to deploy Endpoint and sending data to Fleet and Elasticsearch.

Scenario Outline: Adding the Endpoint Integration to an Agent makes the host to show in Security App
  Given a "<os>" agent is deployed to Fleet with "tar" installer
    And the agent is listed in Fleet as "online"
  When the "Endpoint Security" integration is "added" in the policy
  Then the "Endpoint Security" datasource is shown in the policy as added
    And the host name is shown in the Administration view in the Security App as "online"

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

Scenario Outline: Deploying an Endpoint makes policies to appear in the Security App
  When an "Endpoint" is successfully deployed with a "<os>" Agent using "tar" installer
  Then the policy response will be shown in the Security App

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

Scenario Outline: Changing an Agent policy is reflected in the Security App
  Given an "Endpoint" is successfully deployed with a "<os>" Agent using "tar" installer
  When the policy is updated to have "malware" in "detect" mode
  Then the policy will reflect the change in the Security App

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

Scenario Outline: Un-enrolling Elastic Agent stops Elastic Endpoint
  Given an "Endpoint" is successfully deployed with a "<os>" Agent using "tar" installer
  When the agent is un-enrolled
  Then the agent is listed in Fleet as "inactive"
    And the host name is not shown in the Administration view in the Security App

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |

Scenario Outline: Removing Endpoint from Agent policy stops the connected Endpoint
  Given an "Endpoint" is successfully deployed with a "<os>" Agent using "tar" installer
  When the "Endpoint Security" integration is "removed" in the policy
  Then the agent is listed in Fleet as "online"
    But the host name is not shown in the Administration view in the Security App

@centos
Examples: Centos
| os     |
| centos |

@debian
Examples: Debian
| os     |
| debian |
