@apache
Feature: As a Metricbeat developer I want to check that the Apache module works as expected

Scenario Outline: Check module is sending metrics to Elasticsearch without errors
  Given Apache "<apache_version>" is running for metricbeat
    And metricbeat is installed and configured for Apache module
  Then there are "Apache" events in the index
    And there are no errors in the index
Examples:
| apache_version |
| 2.2            |
| 2.4            |
