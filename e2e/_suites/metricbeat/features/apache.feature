@apache
Feature: Apache
  As a Metricbeat developer I want to check that the Apache module works as expected

Scenario Outline: Apache-<apache_version> sends metrics to Elasticsearch without errors
  Given Apache "<apache_version>" is running for metricbeat
    And metricbeat is installed and configured for Apache module
    And metricbeat waits "20" seconds for the service
  When metricbeat runs for "20" seconds 
  Then there are "Apache" events in the index
    And there are no errors in the index
Examples:
| apache_version |
| 2.4.12         |
| 2.4.20         |
