@vsphere
Feature: As a Metricbeat developer I want to check that the vSphere module works as expected

Scenario Outline: Check module is sending metrics to Elasticsearch without errors
  Given vSphere "<vsphere_version>" is running for metricbeat
    And metricbeat is installed and configured for vSphere module
  When metricbeat runs for "20" seconds after waiting "60" seconds for the service
  Then there are "vSphere" events in the index
    And there are no errors in the index
Examples:
| vsphere_version |
| latest          |
