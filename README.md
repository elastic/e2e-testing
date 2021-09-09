[![Build Status](https://beats-ci.elastic.co/buildStatus/icon?job=e2e-tests%2Fe2e-testing-mbp%2Fmaster)](https://beats-ci.elastic.co/job/e2e-tests/job/e2e-testing-mbp/job/master/)

# End-2-End tests for the Observability projects

This repository contains:

1. a [Go library](./cli/README.md) to provision services in the way of Docker containers. It will provide the services using Docker Compose files.
1. A [test framework](./e2e/README.md) to execute e2e tests for certain Observability projects:
    - [Observability Helm charts](./e2e/_suites/helm):
        - APM Server
        - Filebeat
        - Metricbeat
    - [Kubernetes Autodiscover](./e2e/_suites/kubernetes-autodiscover)
    - [Fleet](./e2e/_suites/fleet)
        - Stand-Alone mode
        - Fleet mode
<<<<<<< HEAD
    - [Metricbeat Integrations](./e2e/_suites/metricbeat)
        - Apache
        - MySQL
        - Redis
        - vSphere
=======
        - and more!

>>>>>>> 0c689c1 (Update README.md (#1558))
1. A [collection of utilities and helpers used in tests](../internal).

> If you want to start writing E2E tests, please jump to our quickstart guide [here](./e2e/QUICKSTART.md).

## Building

This project utilizes `goreleaser` to build the cli binaries for all supported
platforms. Please see [goreleaser installation](https://goreleaser.com/install/)
for instructions on making that available to you.

Once `goreleaser` is installed building the cli is as follows:

```
$ make build
```

This will put the built distribution inside of `dist` in the current working directory.

## Contributing

### pre-commit

This project uses [pre-commit](https://pre-commit.com/) so, after installing it, please install the already configured pre-commit hooks we support, to enable pre-commit in your local git repository:

```shell
$ pre-commit install
pre-commit installed at .git/hooks/pre-commit
```

To understand more about the hooks we use, please take a look at pre-commit's [configuration file](./.pre-commit-config.yml).
