[![Build Status](https://beats-ci.elastic.co/buildStatus/icon?job=e2e-tests%2Fe2e-testing-mbp%2F7.9.x)](https://beats-ci.elastic.co/job/e2e-tests/job/e2e-testing-mbp/job/7.9.x/)

# End-2-End tests for the Observability projects

This repository contains:

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
1. Contract tests for the integration between Fleet and the tests, using [pact.io](https://pact.io/). For further information, please read [CONTRACT_TESTING.md](CONTRACT_TESTING.md).

## Contributing

### pre-commit

This project uses [pre-commit](https://pre-commit.com/) so, after installing it, please install the already configured pre-commit hooks we support, to enable pre-commit in your local git repository:

```shell
$ pre-commit install
pre-commit installed at .git/hooks/pre-commit
```

To understand more about the hooks we use, please take a look at pre-commit's [configuration file](./.pre-commit-confifg.yml).
