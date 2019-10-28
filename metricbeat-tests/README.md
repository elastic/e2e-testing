# Functional tests for Metricbeat Integrations

As described in the [design issue](https://github.com/elastic/observability-dev/issues/187), the main goals for this tool are:

- execute metricbeat module tests in isolation: one module at a time
- verify that the metricbeat module is able to send metrics to an ~~output file~~ Elasticsearch
- allow the verification of both compiled Golang code, and the build artefact (in a Docker image/container format)
- run tests locally in the same manner as in the CI
- improve developer experience (more third-party developers adoption?)
- run tests against the integrations compatibility matrix (MySQL 5.6, 5.7, 8.0, etc)

## Goals

### Execute metricbeat module tests, one module at a time
Because we are decoupling the specs, configuration files, and tests for each module in different files, if desired, we will be able to instrument the test runner to run a module in isolation.

On the contrary, we will be able to run all tests in a single run.

### Verify that metricbeat sends metrics to Elasticsearch
In a previous approach, we simply verified that Metricbeat sent metrics to an output file. Once this POC is more mature, we are able now to verify it sends metrics to an Elasticsearch. At that point, we query Elasticsearch checking that there are no errors in the index.

### Run tests locally
We want the developers to be able to run the tests locally in an easy manner. Using local build tools, like `Make` we are wrapping up the main tasks in these project, so that we must learn just a few commands. If you do not want to use `Make` the test runner uses `go test` under the hood, so it's super easy to understand what's going on in terms of Go context. Our recommendation is to use the `Make` wrapper, or to use the `Godog` test runner.

For the CI, we have a set of scripts preparing the environment, so that a build is totally repeatable.

### Improve Developer Experience
A consequence of the above goal is that it's easier to run the tests, developers are more willing to participate in the development checking that their changes are still valid.

### Run tests against the integrations compatibility matrix
With the Scenario outline approach, where we provide a table of possible values, it's possible to iterate through that table and execute a test per row. So if we are smart enough to build the table in the proper manner, then we will be able to create a compatibility matrix for each version of the integration module.

## Technology stack

As we want to run _functional tests_, we need a manner to describe the functionality to implement in a _functional manner_, which means using plain English to specify how our software behaves. The most accepted manner to achieve this specification in the software industry, using a high level approach that anybody in the team could understand and backed by a testing framework, is [`Cucumber`](https://cucumber.io). So we will use `Cucumber` to set the behaviour (use cases) of our software.

Then we need a manner to connect that plain English feature specification with code. Fortunately, `Cucumber` has a wide number of implementations (Java, Ruby, NodeJS, Go...), so we can choose one of them to implement our tests.

As metricbeat is a Golang project, we are going to use Golang for writing the functional tests, so we would need the Golang implementation for `Cucumber`. That implementation is [`Godog`](https://github.com/DATA-DOG/godog), which is the glue between the specs files and the Go code. Godog is a wrapper over the traditional `go test` command, adding the ability to run the functional steps defined in the feature files.

### Integration Modules
The services supported by Metricbeat integrations will be started in the form of Docker containers. To manage the life cycle of those containers in test time we are going to use [`Testcontainers`](https://testcontainers.org), a set of libraries to simplify the usage of the Docker client, attaching container life cycles to the tests, so whenever the tests finish, the containers will stop in consequence.

### Runtime dependencies
We want to store the metrics in Elasticsearch, so at some point we must start up an Elasticsearch instance. Besides that, we want to query the Elasticsearch to perform assertions on the metrics, such as there are no errors, or the field `f.foo` takes the value `bar`. For that reason we need an Elasticsearch in a well-known location. Here it appears the usage of the [Observability Provisioner CLI tool](../cli/README.md), which is a CLI writen in Go which exposes an API to query the specific runtime resources needed to run the tests. In our case, Metricbeat, we need just an Elasticsearch, but a Kibana could be needed in the case of verifying the dashboards are correct.

## Adding tests for a new module
Ok, you want to contribute the tests for a new integration module. Then you have to simply add three files, that's all. Therefore, a test is formed by three elements:

- A `feature file`, describing in plain English the use cases and test scenarios.
- A `compose file`, in Docker Compose format, with any configuration needed for the module.
- A `configuration file`, in YAML format, with any Metricbeat configuration that is specific to the module.

### Feature files
We will create use cases for the module in a separate `.feature` file, ideally named after module's name (i.e. _apache.feature_). This feature file is a Cucumber requirement, that will be parsed by the test runner and matched against the Golang code implementing the tests.

```cucumber
@apache
Feature: As a Metricbeat developer I want to check that the Apache module works as expected

Scenario Outline: Check module is sending metrics to Elasticsearch without errors
  Given Apache "<apache_version>" is running for metricbeat
    And metricbeat is installed and configured for Apache module
  Then there are "Apache" events in the index
    And there are no errors in the index
Examples:
| apache_version |
| 2.2            |
| 2.4            |
```

>You should write as many scenarios as you considering, covering different use cases in each scenario, taking care of duplicated steps that could be reused by other module.

The anatomy of a feature file is:

- **@apache**: A `@` character indicates a tag. And tags are used to filter the test execution. Tags could be placed on Features (applying the entire file), or on Scenarios (applying just to them). At this moment we are tagging each feature file with a tag using module's name, so that we can instrument the test runner to just run one.
- **Feature: Blah, blah**: Description in plain English of the group of uses cases (scenarios) in this feature file. The feature file should contain just one.
- **Scenario**: the name in plain English of a specific use case.
- **Scenario Outline**: exactly the same as above, but we are are telling Cucumber that this use case has a matrix, so it has to iterate through the **Examples** table, interpolating those values into the placeholders in the scenario.
- **Given, Then, When, And, But keywords**: Their meaning is extremely important in order to understand the use case they are part of, although they have no real impact in how we use them. If we use `doble quotes` around one or more words, that will tell Cucumber the presence of a fixed variable, with value the word/s among the double quotes. These variables will be the input parameters of the implementation functions in Go code. If we use `angles` around one or more words, that will tell Cucumber the presence of a dynamic variable, taken from the examples table.
    - **Given**: It must tell an ocational reader what state must be in place for the use case to be valid.
    - **When**: It must tell an ocational reader what action or actions trigger the use case.
    - **Then**: It must tell an ocational reader what outcome has been generated after the use case happens.
    - **And**: Used within any of the above clauses, it must tell an ocational reader a secondary preparation (Given), trigger (When), or output (Then) that must be present.
    - **But**: Used within any of the above clauses, it must tell an ocational reader a secondary preparation (Given), trigger (When), or output (Then) that must not be present.
- **Examples:**: this `markdown table` will represent the elements to interpolate in the existing dynamic variables in the use case, being each column header the name of the different variables in the table. Besides that, each row will result in a test execution.

### Configuration files
There will exist a configuration YAML file per module, under the `configurations` folder. The name of the file will be the module name (i.e. `apache.yml`). In this file we will add those configurations that are exclusive to the module, as those that are common are already defined at Metricbeat level.

## Running the tests
At this moment, the CLI and the functional tests coexist in the same repository, that's why we are building the CLI to get access to its features. Eventually that would change and we would consume it as a binary. Meanwhile, execute this from the ROOT directory of this project:

```shell
$ export STACK_VERSION=7.5.0                       # exports stack version as runtime
$ export METRICBEAT_VERSION=7.5.0                  # exports metricbeat version to be tested
$ # export FEATURE=redis                           # exports which feature to run (default 'all')
$ make -C metricbeat-tests build-binary            # generates the binary from the repository
$ make -C metricbeat-tests run-elastic-stack       # runs the stack for metricbeat
$ make -C metricbeat-tests functional-test         # runs the test suite for Redis and stack 
$ make -C metricbeat-tests shutdown-elastic-stack  # stops the stack
```

You could set up the environment so that it's possible to run one single module. As we are using _tags_ for matching modules, we could tell `make` to run just the tests for redis:

```shell
$ LOG_LEVEL=DEBUG FEATURE="redis" make functional-test
```
where:

- LOG_LEVEL: sets the default log level in the tool (DEBUG, INFO, WARN, ERROR, FATAL)
- FEATURE: sets the tag to filter by (apache, mysql, redis)

### Advanced usage
There are some environment variables you can use to improve the experience running the tests with `Make`.

- **METRICBEAT_FETCH_TIMEOUT** (default: 20). This is the time in seconds we leave metricbeat grabbing metrics from the monitored integration module.
- **QUERY_MAX_ATTEMPTS** (default: 5). It's possible that the Elasticsearch is not ready for writes, so we can define a retry strategy to wait for our index to be ready. This variable defines the number of attempts the retry process will happen.
- **RETRY_TIMEOUT** (default: 3). For same reason as above, this variable defines the delay between attempts, before a successful connection to Elasticsearch is made.

>Interested in running the tests directly using Godog? Please check out [the Makefile](./Makefile#L19).

```shell
export OP_METRICBEAT_FETCH_TIMEOUT=${OP_METRICBEAT_FETCH_TIMEOUT:-20}
export OP_QUERY_MAX_ATTEMPTS=${OP_QUERY_MAX_ATTEMPTS:-5}
export OP_RETRY_TIMEOUT=${OP_RETRY_TIMEOUT:-3}
export FORMAT=${FORMAT:-pretty} # valid formats are: pretty, junit
# If you do not pass a '-t moduleName' argument, then all tests will be run
godog --format=${FORMAT} -t redis
```

>For environment variables reference affecting the logs, please check out [CLI's docs](../cli/README.md#logging)
