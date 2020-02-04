@mysql
Feature: As a Metricbeat developer I want to check that the MySQL module works as expected

Scenario Outline: Check module is sending metrics to Elasticsearch without errors
  Given MySQL "<mysql_version>" is running for metricbeat
    And metricbeat is installed and configured for MySQL module
    And metricbeat waits "20" seconds for the service
  When metricbeat runs for "20" seconds
  Then there are "<variant>" events in the index
    And there are no errors in the index
Examples:
| mysql_version      | variant |
| mariadb-10.2.23-1  | MariaDB |
| mariadb-10.3.14-1  | MariaDB |
| mariadb-10.4.4-1   | MariaDB |
| mysql-5.7.12-1     | MySQL   |
| mysql-8.0.13-1     | MySQL   |
| percona-5.7.24-1   | Percona |
| percona-8.0.13-4-1 | Percona |
