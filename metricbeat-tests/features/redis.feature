@redis
Feature: As a Metricbeat developer I want to check that the Redis module works as expected

Scenario Outline: Check module is sending metrics to a file
  Given Redis "<redis_version>" is running
    And metricbeat "<metricbeat_version>" is installed and configured for Redis module
  Then metricbeat stores metrics to elasticsearch in the index "metricbeat-<metricbeat_version>"
Examples:
| redis_version | metricbeat_version |
| 4.0  | 7.3.0 |
| 4.0  | 8.0.0-SNAPSHOT |
| 5.0  | 7.3.0 |
| 5.0  | 8.0.0-SNAPSHOT |
