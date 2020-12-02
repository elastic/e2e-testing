@integrations
Feature: Integrations
  As a Metricbeat developer I want to check that the Integrations modules work as expected

Scenario Outline: <integration>-<version> sends metrics to Elasticsearch without errors
  Given "<integration>" "<version>" is running for metricbeat
  When metricbeat is installed and configured for "<integration>" module
  Then there are "<integration>" events in the index
    And there are no errors in the index

@apache
Examples:
| integration | version |
| apache      | 2.4.12  |
| apache      | 2.4.20  |

@redis
Examples:
| integration | version |
| redis       | 3.2.12  |
| redis       | 4.0.11  |
| redis       | 5.0.5   |

@vsphere
Examples:
| integration | version |
| vsphere     | latest  |
