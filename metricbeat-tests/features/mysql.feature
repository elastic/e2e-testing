@mysql
Feature: As a Metricbeat developer I want to check that the MySQL module works as expected

Scenario Outline: Check module is sending metrics to Elasticsearch without errors
  Given MySQL "<mysql_version>" is running for metricbeat
    And metricbeat is installed and configured for MySQL module
    And metricbeat waits "20" seconds for the service
  When metricbeat runs for "20" seconds
  Then there are "MySQL" events in the index
    And there are no errors in the index
Examples:
| mysql_version  |
| mysql-5.7.12-1 |
| mysql-8.0.13-1 |
