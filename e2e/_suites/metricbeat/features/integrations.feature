@integrations
Feature: Integrations
  As a Metricbeat developer I want to check that the Integrations modules work as expected

Scenario Outline: <integration>-<version> sends metrics to Elasticsearch without errors
  Given "<integration>" "<version>" is running for metricbeat
  When metricbeat is installed and configured for "<integration>" module
  Then there are "<integration>" events in the index
    And there are no errors in the index

@apache
Examples: Apache
| integration | version |
| apache      | 2.4.12  |
| apache      | 2.4.20  |

@dropwizard
Examples: Dropwizard
| integration | version |
| dropwizard  | 4.1.2   |
| dropwizard  | 4.0.0   |
| dropwizard  | 3.2.6   |

@envoyproxy
Examples: EnvoyProxy
| integration | version |
| envoyproxy  | 1.7.0   |
| envoyproxy  | 1.12.0  |

@etcd
Examples: Etcd
| integration | version |
| etcd        | 3.2.25  |
| etcd        | 3.3.10  |

@haproxy
Examples: HAProxy
| integration | version |
| haproxy     | 2.0.9   |
| haproxy     | 1.9.12  |
| haproxy     | 1.8.22  |
| haproxy     | 1.7.12  |
| haproxy     | 1.6.15  |

@redis
Examples: Redis
| integration | version |
| redis       | 3.2.12  |
| redis       | 4.0.11  |
| redis       | 5.0.5   |

@vsphere
Examples: vSphere
| integration | version |
| vsphere     | latest  |

@variants
Scenario Outline: <integration>-<variant>-<version> sends metrics to Elasticsearch without errors
  Given "<variant>" v<version>, variant of "<integration>", is running for metricbeat
  When metricbeat is installed and configured for "<variant>", variant of the "<integration>" module
  Then there are "<variant>" events in the index
    And there are no errors in the index

@mysql
Examples: MySQL
| integration | variant | version  |
| mysql       | MariaDB | 10.2.23  |
| mysql       | MariaDB | 10.3.14  |
| mysql       | MariaDB | 10.4.4   |
| mysql       | MySQL   | 5.7.12   |
| mysql       | MySQL   | 8.0.13   |
| mysql       | Percona | 5.7.24   |
| mysql       | Percona | 8.0.13-4 |
