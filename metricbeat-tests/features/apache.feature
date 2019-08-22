@apache
Feature: As a Metricbeat developer I want to check that the Apache module works as expected

Scenario Outline: Check module is sending metrics to a file
  Given Apache "<apache_version>" is running
    And metricbeat "<metricbeat_version>" is installed and configured for Apache module
  Then metricbeat stores metrics to elasticsearch in the index "metricbeat-<metricbeat_version>"
Examples:
| apache_version | metricbeat_version |
| 2.2  | 7.3.0 |
| 2.2  | 8.0.0-SNAPSHOT |
| 2.4  | 7.3.0 |
| 2.4  | 8.0.0-SNAPSHOT |
