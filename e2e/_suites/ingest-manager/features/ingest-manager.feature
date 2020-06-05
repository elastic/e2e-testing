@ingest
Feature: Enable Fleet user and create initial Kibana setup

@enroll
Scenario: Enrolling an agent
  Given there is a "Fleet" user in Kibana
    And the "Fleet" Kibana setup has been created
  When the agent binary is installed in the target host
  Then the dashboards for the agent are present in Elasticsearch
    And the agent shows up in Kibana

@unenroll
Scenario: Un-enrolling an agent
  Given an agent is enrolled
  When the agent is un-enrolled from Kibana
  Then no new data shows up in Elasticsearc locations using the enrollment token

@reenroll
Scenario: Enrolling, un-enrolling and re-enrolling an agent
  Given an agent is enrolled
    And the agent is un-enrolled from Kibana
  When the agent is re-enrolled from the host
    And the agent runs from the host
  Then the agent shows up in Kibana

@revoke-token
Scenario: Revoking the enrollment token for an agent
  Given an agent is enrolled
  When the enrollment token is revoked
  Then it's not possible to use the token to enroll more agents
    And the enrolled agent continues to work

@start-agent
Scenario: Starting the agent starts backend processes
  When the agent is started in the host
  Then filebeat is started
    And metricbeat is started
    And endpoint is started

@stop-agent
Scenario: Stopping the agent stops backend processes
  Given an agent is started in the host
  When the agent is stopped in the host
  Then filebeat is stopped
    And metricbeat is stopped
    And endpoint is stopped
