@metricbeat
Feature: As a Metricbeat developer I want to check that default configuration works as expected

Scenario Outline: Check <configuration> configuration is sending metrics to Elasticsearch without errors
  Given metricbeat "<metricbeat_version>" is installed using "<configuration>" configuration
  Then there are "system" events in the index
    And there are no errors in the index
Examples:
| metricbeat_version | configuration        |
| 7.3.0              | metricbeat           |
| 7.4.0              | metricbeat           |
| 8.0.0-SNAPSHOT     | metricbeat           |
| 7.3.0              | metricbeat.docker    | 
| 7.4.0              | metricbeat.docker    |
| 8.0.0-SNAPSHOT     | metricbeat.docker    |
| 7.3.0              | metricbeat.reference | 
| 7.4.0              | metricbeat.reference |
| 8.0.0-SNAPSHOT     | metricbeat.reference |
