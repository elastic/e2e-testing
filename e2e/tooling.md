# Tooling
We need a manner to describe the functionality to implement in a _functional manner_. And it would be great if we are able to use plain English to specify how our software behaves, instead of using code. And if it's possible to automate the execution of that specification, even better.

The most accepted manner to achieve this executable specification in the software industry, using a high level approach that everybody in the team could understand and backed by a testing framework, is [`Cucumber`](https://cucumber.io). So we will use `Cucumber` to set the behaviours (use cases) of our software.

Then we need a manner to connect that plain English feature specification with code. Fortunately, `Cucumber` has a wide number of implementations (Java, Ruby, NodeJS, Go...), so we can choose one of them to implement our tests.

We are going to use Go for writing the End-2-End tests, so we would need the Go implementation for `Cucumber`. That implementation is [`Godog`](https://github.com/cucumber/godog), which is the glue between the spec files and the Go code. `Godog` is a wrapper over the `go test` command, so they are almost interchangeable when running the tests.

> In this test framework, we are running Godog with `go test`, as explained [here](https://github.com/cucumber/godog#running-godog-with-go-test).

## Cucumber: BDD at its core
The specification of these E2E tests has been done using `BDD` (Behaviour-Driven Development) principles, where:

>BDD aims to narrow the communication gaps between team members, foster better understanding of the customer and promote continuous communication with real world examples.

From Cucumber's website:

>Cucumber is a tool that supports Behaviour-Driven Development(BDD), and it reads executable specifications written in plain text and validates that the software does what those specifications say. The specifications consists of multiple examples, or scenarios.

The way we are going to specify our software is using [`Gherkin`](https://cucumber.io/docs/gherkin/reference/).

>Gherkin uses a set of special keywords to give structure and meaning to executable specifications. Each keyword is translated to many spoken languages. Most lines in a Gherkin document start with one of the keywords.

The key part here is **executable specifications**: we will be able to automate the verification of the specifications and potentially get a coverage of these specs.

## Godog: Cucumber for Go
From Godog's website:

>Package godog is the official Cucumber BDD framework for Go, it merges specification and test documentation into one cohesive whole.

For this test framework, we have chosen Godog over any other test framework because the Beats team (the team we started working with) is already using Go, so it seems reasonable to choose it.

## Build system
The test framework makes use of `Make` to prepare the environment to run the `Cucumber` tests. Although it's still possible using the `godog` binary, or the idiomatic `go test`, to run the test suites and scenarios, we recommend using the proper `Make` goals, in particular the [`functional-test` goal](https://github.com/elastic/e2e-testing/blob/05cdf195dfae0cb886d30b1cf6a1ccd95ddba9f5/e2e/commons-test.mk#L56). We also provide [a set of example goals](https://github.com/elastic/e2e-testing/blob/05cdf195dfae0cb886d30b1cf6a1ccd95ddba9f5/e2e/Makefile#L22-L35) with different use cases for running most common scenarios.

Each test suite, which lives under the `e2e/_suites` directory, has it's own Makefile to control the build life cycle of the test project.

> It's possible to create a new test suite with `SUITE=name make -C e2e create-suite`, which creates the build files and the scaffolding for the first test suite.

## Runtime dependencies
In many cases, we want to store the metrics in Elasticsearch, so at some point we must start up an Elasticsearch instance. Besides that, we want to query the Elasticsearch to perform assertions on the metrics, such as there are no errors, or the field `f.foo` takes the value `bar`. For that reason we need an Elasticsearch instance in a well-known location. We are going to group this Elasticsearch instance, and any other runtime dependencies, under the concept of a `profile`, which is a represented by a `docker-compose.yml` file under the `cli/config/compose/profiles/` and the name of the test suite.

As an example, the Fleet test suite will need an Elasticsearch instance, Kibana and Fleet Server.

## Generating documentation about the specifications
If you want to transform your feature files into a nicer representation using HTML, please run this command from the root `e2e` directory to build a website for all test suites:

```shell
$ make build-docs
```

It will generate the website under the `./docs` directory (which is ignored in Git). You'll be able to navigate through any feature file and test scenario in a website.
