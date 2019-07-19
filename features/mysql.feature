Feature: As a Metricbeat developer I want to check that the MySQL module works as expected

Scenario Outline: Check module is sending metrics to a file
  Given MySQL "<mysql_version>" is running on port "<mysql_port>"
    And metricbeat "7.2.0" is installed and configured for MySQL module
  Then metricbeat outputs metrics to the file "mysql-<mysql_version>.metrics"
Examples:
| mysql_version | mysql_port |
| 5.6  | 3306 |
| 5.7  | 3307 |
| 8.0  | 3308 |
