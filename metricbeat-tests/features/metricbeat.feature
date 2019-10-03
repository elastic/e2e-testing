@metricbeat
Feature: As a Metricbeat developer I want to check that default configuration works as expected

Scenario Outline: Check default configuration is sending metrics to Elasticsearch without errors
  Given metricbeat "<metricbeat_version>" is installed using default configuration
  Then there are "system" events in the index
    And there are no errors in the index
Examples:
| metricbeat_version |
| 7.3.0 |
| 7.4.0 |
| 8.0.0-SNAPSHOT |
