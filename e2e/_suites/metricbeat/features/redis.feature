@redis
Feature: Redis
  As a Metricbeat developer I want to check that the Redis module works as expected

Scenario Outline: Redis-<redis_version> sends metrics to Elasticsearch without errors
  Given Redis "<redis_version>" is running for metricbeat
  When metricbeat is installed and configured for Redis module
  Then there are "Redis" events in the index
    And there are no errors in the index
Examples:
| redis_version |
| 3.2.12        |
| 4.0.11        |
| 5.0.5         |
