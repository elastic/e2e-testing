[![Build Status](https://beats-ci.elastic.co/buildStatus/icon?job=e2e-tests%2Fe2e-testing-mbp%2Fmain)](https://beats-ci.elastic.co/job/e2e-tests/job/e2e-testing-mbp/job/main/)

# End-2-End tests for the Observability projects

This repository contains:

1. A [CI Infrastructure](./.ci/README.md) to provision VMs where the tests will be executed at CI time.
2. A [Go library](./cli/README.md) to provision services in the way of Docker containers. It will provide the services using Docker Compose files.
3. A [test framework](./e2e/README.md) to execute e2e tests for certain Observability projects:
    - [Kubernetes Autodiscover](./e2e/_suites/kubernetes-autodiscover)
    - [Fleet](./e2e/_suites/fleet)
        - Stand-Alone mode
        - Fleet mode
4. A [collection of utilities and helpers used in tests](../internal).
        - and more!

> If you want to start creating a new test suite, please read [the quickstart guide](./QUICKSTART.md), but don't forget to come back here to better understand the framework.

> If you want to start running the tests, please read [the "running the tests" guide](./.ci/README.md).

The E2E test project uses `BDD` (Behavioral Driven Development), which means the tests are defined as test scenarios (or simply `scenarios`). A scenario is written in **plain English**, using business language hiding any implementation details. Therefore, the words _"clicks", "input", "browse",  "API call"_ are NOT allowed. And we do care about having well-expressed language in the feature files. Why? Because we want to hide the implementation details in the tests, and whoever is reading the feature files is able to understand the expected behavior of each scenario. And for us that's the key when talking about real E2E tests: to exercise user journeys (scenarios) instead of specific parts of the UI (graphical or API).

## Behaviour-Driven Development and this test framework

We need a manner to describe the functionality to be implemented _in a functional manner_. And it would be great if we are able to use plain English to specify how our software behaves, instead of using code. And if it's possible to automate the execution of that specification, even better. These behaviours of our software, to be consistent, they must be implemented following certain `BDD` (Behaviour-Driven Development) principles, where:

> BDD aims to narrow the communication gaps between team members, foster better understanding of the customer and promote continuous communication with real world examples.

The most accepted manner to achieve this executable specification in the software industry, using a high level approach that everybody in the team could understand and backed by a testing framework to automate it, is [`Cucumber`](https://cucumber.io). So we will use `Cucumber` to set the behaviours (use cases) for our software. From its website:

> Cucumber is a tool that supports Behaviour-Driven Development(BDD), and it reads executable specifications written in plain text and validates that the software does what those specifications say. The specifications consists of multiple examples, or scenarios.

The way we are going to specify our software behaviours is using [`Gherkin`](https://cucumber.io/docs/gherkin/reference/):

>Gherkin uses a set of special keywords to give structure and meaning to executable specifications. Each keyword is translated to many spoken languages. Most lines in a Gherkin document start with one of the keywords.

The key part here is **executable specifications**: we will be able to automate the verification of the specifications and potentially get a coverage of these specs.

Then we need a manner to connect that plain English feature specification with code. Fortunately, `Cucumber` has a wide number of implementations (Java, Ruby, NodeJS, Go...), so we can choose one of them to implement our tests. For this test framework, we have chosen [Godog](https://github.com/cucumber/godog), the Go implementation for Cucumber. From its website:

> Package godog is the official Cucumber BDD framework for Go, it merges specification and test documentation into one cohesive whole.

In this test framework, we are running Godog with `go test`, as explained [here](https://github.com/cucumber/godog#running-godog-with-go-test).

## Statements about the test framework

- It uses the `Given/When/Then` pattern to define scenarios.
  - `Given` a state is present (optional)
  - `When` an action happens (mandatory) 
  - `Then` an outcome is expected (mandatory)
  - `And/But` clauses are allowed to provide continuation on the above ones. (Optional)
- Because it uses BDD, it's possible to combine existing scenario steps (the given-when-then clauses) forming new scenarios, so with no code it's possible to have new tests. For that reason, the steps must be atomic and decoupled (as any piece of software: low coupling and high cohesion).
- It does not use the GUI at all, so there is no Selenium, Cypress or any other test library having expectations on graphical components. It uses Go as the programming language and Cucumber to create the glue code between the code and the plain English scenarios. Over time, we have demonstrated that the APIs are not changing as fast as the GUI components.
- APIs not changing does not mean zero flakiness. Because there are so many moving pieces (stack versions), beats versions, elastic-agent versions, cloud machines, network access, etc... There could be situations where the tests fail, but they are rarely caused by test flakiness. Instead, they are usually caused by: 1) instability of the all-together system, 2) a real bug, 3) gherkin steps that are not consistent.
- Kibana is basically at the core of the tests, because we hit certain endpoints and wait for the responses.
  - Yes, the e2e tests are first citizen consumers of Kibana APIs, so they could be broken at the moment an API change on Kibana. We have explored the idea of implementing `Contract-Testing` with [pact.io](https://pact.io) (not implemented but in the wish list).
  - A PoC was submitted to [kibana](https://github.com/elastic/kibana/pull/80384) and to [this repo](https://github.com/elastic/e2e-testing/pull/339) demonstrating the benefits of `Contract-Testing`.
- The project usually checks for JSON responses, OS processes state, Elasticsearch queries responses (using the ES client), to name a few, so the majority of the assertions relies on checking those entities and its internal state: process state, JSON response, HTTP codes, document count in an Elasticsearch query, etc.

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
