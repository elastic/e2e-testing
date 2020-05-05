Feature: Parity Tests

Scenario Outline: The <product> documents indexed by the legacy collection method are identical in structure to those indexed by Metricbeat collection
  Given "<product>" sends metrics to Elasticsearch using the "legacy" collection monitoring method
  When "<product>" sends metrics to Elasticsearch using the "metricbeat" collection monitoring method
  Then the structure of the documents for the "legacy" and "metricbeat" collection are identical
Examples:
| product       |
| elasticsearch |
| kibana        |
| logstash      |
| filebeat      |
