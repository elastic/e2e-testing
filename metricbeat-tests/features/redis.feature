@redis
Feature: As a Metricbeat developer I want to check that the Redis module works as expected

Scenario Outline: Check module is sending metrics to Elasticsearch
  Given Redis "<redis_version>" is running
    And metricbeat "<metricbeat_version>" is installed and configured for Redis module
  Then there are no errors in the "metricbeat-<metricbeat_version>" index
Examples:
| redis_version | metricbeat_version |
| 4.0.14  | 7.3.0 |
| 4.0.14  | 8.0.0-SNAPSHOT |
| 5.0.5  | 7.3.0 |
| 5.0.5  | 8.0.0-SNAPSHOT |
