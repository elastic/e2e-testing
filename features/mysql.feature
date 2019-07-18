Feature: As a Metricbeat developer I want to check that the MySQL module works as expected

Scenario Outline: Check module is sending metrics to a file
  Given MySQL "<mysql_version>" is running
    And metricbeat "7.2.0" is installed and configured for MySQL module
  Then metricbeat outputs metrics to the file "metricbeat-mysql-<mysql_version>.metrics"
Examples:
| mysql_version |
| 5.6  |
| 5.7  |
| 8.0  |
