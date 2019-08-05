@mysql
Feature: As a Metricbeat developer I want to check that the MySQL module works as expected

Scenario Outline: Check module is sending metrics to a file
  Given MySQL "<mysql_version>" is running
    And metricbeat "<metricbeat_version>" is installed and configured for MySQL module
  Then metricbeat outputs metrics to the file "metricbeat-<metricbeat_version>-mysql-<mysql_version>.metrics"
Examples:
| mysql_version | metricbeat_version |
| 5.6  | 7.3.0 |
| 5.7  | 7.3.0 |
| 8.0  | 7.3.0 |
| 5.6  | 8.0.0-SNAPSHOT |
| 5.7  | 8.0.0-SNAPSHOT |
| 8.0  | 8.0.0-SNAPSHOT |
