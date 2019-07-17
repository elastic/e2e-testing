# Formal verification of Metricbeat module

We want to make sure that a metricbeat module works as expected, taking into
consideration different versions of the integration software that metricbeats uses.
So for that reason we are adding [smoke tests](http://softwaretestingfundamentals.com/smoke-testing/) to verify that the redeployment has been done with certain grade of satisfaction.

>Smoke Testing, also known as “Build Verification Testing”, is a type of software testing that comprises of a non-exhaustive set of tests that aim at ensuring that the most important functions work. The result of this testing is used to decide if a build is stable enough to proceed with further testing.

## Running the tests

The tests are located under the [./](root) directory. Place your terminal there and execute `godog`, which is a tool developed by DATADOG folks supporting Cucumber:

```shell
$ godog
```

## Tooling

The specification of these smoke tests has been done using the `BDD` (Behaviour-Driven Development) principles, where:

>BDD aims to narrow the communication gaps between team members, foster better understanding of the customer and promote continuous communication with real world examples.

The implementation of these smoke tests has been done with [Godog](https://github.com/DATA-DOG/godog) + [Cucumber](https://cucumber.io/).

### Cucumber: BDD at its core

From their website:

>Cucumber is a tool that supports Behaviour-Driven Development(BDD), and it reads executable specifications written in plain text and validates that the software does what those specifications say. The specifications consists of multiple examples, or scenarios.

The way we are going to specify our software is using [`Gherkin`](https://cucumber.io/docs/gherkin/reference/).

>Gherkin uses a set of special keywords to give structure and meaning to executable specifications. Each keyword is translated to many spoken languages. Most lines in a Gherkin document start with one of the keywords.

The key part here is **executable specifications**: we will be able to automate the verification of the spefications anf potentially get a coverage of these specs.

### Godog: Cucumber for Golang

From their website:

>Package godog is the official Cucumber BDD framework for Golang, it merges specification and test documentation into one cohesive whole.

For this POC, we have chosen Godog over any other test framework because the team is using already using Golang, so it seems reasonable to choose it.

## Test Specification

All the Gherkin (Cucumber) specifications are written in `.feature` files.

A good example could be [this one](./features/mysql.feature):

```cucumber
Feature: As a Metricbeat developer I want to check that the MySQL module works as expected

Scenario Outline: Check module is sending metrics to elasticsearch
  Given metricbeat "7.2.0" is installed and configured for MySQL module
    And MySQL "<mysql_version>" is running
  Then metricbeat outputs metrics to the file "metricbeat-mysql-<mysql_version>.metrics"
Examples:
| mysql_version |
| 5.6  |
| 5.7  |
| 8.0  |
```

## Test Implementation

We are using Godog + Cucumber to implement the tests, where we create connections to the `Given`, `When`, `Then`, `And`, etc. in a well-known file structure.

As an example, the Golang implementation of the `features/mysql.feature` is located under the [./mysql_test.go](./mysql_test.go) file.