@mysql
Feature: Mysql
  As a Metricbeat developer I want to check that the MySQL module works as expected

Scenario Outline: <variant>-<version> sends metrics to Elasticsearch without errors
  Given "<variant>" v<version>, variant of "MySQL", is running for metricbeat
  When metricbeat is installed and configured for "<variant>", variant of the "MySQL" module
  Then there are "<variant>" events in the index
    And there are no errors in the index
Examples:
| variant | version    |
| MariaDB | 10.2.23    |
| MariaDB | 10.3.14    |
| MariaDB | 10.4.4     |
| MySQL   | 5.7.12     |
| MySQL   | 8.0.13     |
| Percona | 5.7.24     |
| Percona | 8.0.13-4   |
