@metricbeat
Feature: Metricbeat
  As a Metricbeat developer I want to check that default configuration works as expected

Scenario Outline: Metricbeat's <configuration> configuration sends metrics to Elasticsearch without errors
  When metricbeat is installed using "<configuration>" configuration
  Then there are "system" events in the index
    And there are no errors in the index
Examples:
| configuration        |
| metricbeat           |
| metricbeat.docker    |
