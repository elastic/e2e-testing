@metricbeat
Feature: Metricbeat
  As a Metricbeat developer I want to check that default configuration works as expected

Scenario Outline: Check <configuration> configuration is sending metrics to Elasticsearch without errors
  Given metricbeat is installed using "<configuration>" configuration
  When metricbeat runs for "30" seconds
  Then there are "system" events in the index
    And there are no errors in the index
Examples:
| configuration        |
| metricbeat           |
| metricbeat.docker    |
