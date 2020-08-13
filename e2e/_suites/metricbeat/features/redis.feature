@redis
Feature: Redis
  As a Metricbeat developer I want to check that the Redis module works as expected

Scenario Outline: Check Redis-<redis_version> is sending metrics to Elasticsearch without errors
  Given Redis "<redis_version>" is running for metricbeat
    And metricbeat is installed and configured for Redis module
    And metricbeat waits "20" seconds for the service
  When metricbeat runs for "20" seconds
  Then there are "Redis" events in the index
    And there are no errors in the index
Examples:
| redis_version |
| 3.2.12        |
| 4.0.11        |
| 5.0.5         |
