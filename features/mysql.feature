Feature: As a Metricbeat developer I want to check that the MySQL module works as expected

Scenario Outline: Check module is sending metrics to elasticsearch
  Given an Elastic stack in version "7.2.0" is running
  And metricbeat "7.2.0" is installed and configured for MySQL module
  Then I want to check that it's working for MySQL "<mysql_version>"
Examples:
| mysql_version |
| 5.6  |
| 5.7  |
| 8.0  |
