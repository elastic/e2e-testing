Feature: As a Metricbeat developer I want to check that the Apache module works as expected

Scenario Outline: Check module is sending metrics to a file
  Given Apache "<apache_version>" is running on port "80"
    And metricbeat "7.2.0" is installed and configured for Apache module
  Then metricbeat outputs metrics to the file "apache-<apache_version>.metrics"
Examples:
| apache_version |
| 2.2  |
| 2.4  |
