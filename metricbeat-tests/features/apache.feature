@apache
Feature: As a Metricbeat developer I want to check that the Apache module works as expected

Scenario Outline: Check module is sending metrics to Elasticsearch
  Given Apache "<apache_version>" is running for metricbeat "<metricbeat_version>"
    And metricbeat "<metricbeat_version>" is installed and configured for Apache module
  Then there are no errors in the index
Examples:
| apache_version | metricbeat_version |
| 2.2  | 7.3.0 |
| 2.2  | 8.0.0-SNAPSHOT |
| 2.4  | 7.3.0 |
| 2.4  | 8.0.0-SNAPSHOT |
