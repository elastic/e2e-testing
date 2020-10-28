@vsphere
Feature: vSphere
  As a Metricbeat developer I want to check that the vSphere module works as expected

Scenario Outline: vSphere-<vsphere_version> sends metrics to Elasticsearch without errors
  Given vSphere "<vsphere_version>" is running for metricbeat
  When metricbeat is installed and configured for vSphere module
  Then there are "vSphere" events in the index
    And there are no errors in the index
Examples:
| vsphere_version |
| latest          |
