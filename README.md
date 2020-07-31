[![Build Status](https://beats-ci.elastic.co/buildStatus/icon?job=e2e-tests%2Fe2e-testing-mbp%2Fmaster)](https://beats-ci.elastic.co/job/e2e-tests/job/e2e-testing-mbp/job/master/)

# End-2-End tests for the Observability projects

This repository contains

1. a [Go library](./cli/README.md) to provision services in the way of Docker containers. It will provide the services using Docker Compose files.
1. A [test framework](./e2e/README.md) to execute e2e tests for certain Observability projects:
    - [Observability Helm charts](./e2e/_suites/helm):
        - APM Server
        - Filebeat
        - Metricbeat
    - [Ingest Manager](./e2e/_suites/ingest-manager)
        - Stand-Alone mode
        - Fleet mode
    - [Metricbeat Integrations](./e2e/_suites/metricbeat)
        - Apache
        - MySQL
        - Redis
        - vSphere
