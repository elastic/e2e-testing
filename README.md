[![Build Status](https://beats-ci.elastic.co/buildStatus/icon?job=e2e-tests%2Fe2e-testing-mbp%2Fmain)](https://beats-ci.elastic.co/job/e2e-tests/job/e2e-testing-mbp/job/main/)

# End-2-End tests for the Observability projects

This repository contains:

1. A [CI Infrastructure](./.ci/README.md) to provision VMs where the tests will be executed at CI time.
2. A [Go library](./cli/README.md) to provision services in the way of Docker containers. It will provide the services using Docker Compose files.
3. A [test framework](./e2e/README.md) to execute e2e tests for certain Observability projects:
    - [Observability Helm charts](./e2e/_suites/helm):
        - APM Server
        - Filebeat
        - Metricbeat
    - [Kubernetes Autodiscover](./e2e/_suites/kubernetes-autodiscover)
    - [Fleet](./e2e/_suites/fleet)
        - Stand-Alone mode
        - Fleet mode
        - and more!

4. A [collection of utilities and helpers used in tests](../internal).

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

## Backports

This project requires backports to the existing active branches. Those branches are defined in the `.backportrc.json` and `.mergify.yml` files. In order to do so,
there are two different approaches:

### Mergify 🥇

This is the preferred approach. Backports are created automatically as long as the rules defined in [.mergify.yml](.mergify.yml) are fulfilled. From the user's point of
view it's required only to attach a labels to the pull request that should be backported, and once it gets merged the automation happens under the hood.

### Backportrc 👴

This is the traditional approach where the backports are created by the author who created the original pull request. For such, it's required to install
[backport](https://www.npmjs.com/package/backport) and run the command in your terminal

```bash
$ backport  --label <YOUR_LABELS> --auto-assign --pr <YOUR_PR>
```

### Generating documentation about the specifications
If you want to transform your feature files into a nicer representation using HTML, please run this command from the root `e2e` directory to build a website for all test suites:

```shell
$ make build-docs
```

It will generate the website under the `./docs` directory (which is ignored in Git). You'll be able to navigate through any feature file and test scenario in a website.

### Noticing the test framework
To generate the notice files for this project:

1. Execute `make notice` to generate NOTICE.txt file.

## Contributors and maintenance

We have received contributions from multiple teams in different aspects of the e2e tests project, so we are ecstatic to receive them:

  - Adam Stokes and Manuel de la Peña, from the Observability Robots team have created the tests framework.
  - Julia Bardi and Nicolas Chaulet, from Fleet team, frontend engineers, have contributed a few scenarios for Fleet.
  - Eric Davis, QA engineer, has helped in the definition of the scenarios for Fleet.
  - Igor Guz, QA engineer in the Security team, has contributed scenarios for the security-related integrations, such as Endpoint, Linux and System.
  - Christos Markou and Jaime Soriano have contributed the k8s-autodiscover test suite, which is maintained by @elastic/obs-cloudnative-monitoring.
  - Julien Lind, from Fleet, has helped in defining the support matrix in terms of what OSs and architectures need to be run for Fleet test suite.
  - Julien Mailleret, from Infra, has contributed to the Helm charts test suite.
  - Anderson Queiroz (Elastic Agent) and Víctor Martínez (Observability Robots), are currently working on the MacOS support for running the tests on real Apple machines using Elastic's Orka provisioner.
