@redis
Feature: As a Metricbeat developer I want to check that the Redis module works as expected

Scenario Outline: Check module is sending metrics to Elasticsearch without errors
  Given Redis "<redis_version>" is running for metricbeat
    And metricbeat is installed and configured for Redis module
  Then there are "Redis" events in the index
    And there are no errors in the index
Examples:
| redis_version |
| 4.0.14        |
| 5.0.5         |
