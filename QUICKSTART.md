# Quickstart

First, we need to understand how the tests work, and what dependencies we need to run them.

## Dependencies

- Go: Install Go, using the language version defined in the `.go-version` file at the root directory. We recommend using [GVM](https://github.com/andrewkroh/gvm), same as done in the CI, which will allow you to install multiple versions of Go, setting the Go environment in consequence: i.e. `eval "$(gvm 1.17)"`.
- Docker: Docker is needed to run certain build commands that wrap certain build tools. To install and configure it, please read [this guide](https://docs.docker.com/engine/install/).

## How do the tests work?

At the topmost level, the test framework uses a BDD framework written in Go, where we set the expected behavior of use cases in a feature file using Gherkin, and implementing the steps in Go code. The provisining of the services forming the stack is accomplished using Docker Compose and the [testcontainers-go](https://github.com/testcontainers/testcontainers-go) library.

The tests will follow this general high-level approach:

1. Install the runtime dependencies as Docker containers via Docker Compose (the Elastic Stack), happening before the test suite runs. These runtime dependencies are defined in a specific `profile` for Fleet, in the form of a `docker-compose.yml` file. You can find the Fleet profile and its configuration files [here](./internal/config/compose/profiles/fleet).
1. Execute BDD steps representing each scenario. Each step will return an Error if the behavior is not satisfied, marking the step and the scenario as failed, or will return `nil`.

## Adding a new test suite

Let's walk through a quick example to see how to start working with the e2e-testing framework, adding a new test suite:
### Step 1 - Install test depedencies

Godog and other test-related binaries will be installed in their supported versions when the project is first built, thanks to Go modules and Go build system.

### Step 2 - Create a new test suite

Given we create a new test suite **foo** under the "e2e/_suites" workspace. - `SUITE=foo make -C e2e create-suite`

From now on, this is our work directory - `cd e2e/_suites/foo`

### Step 3 - Verify the generated files

We have created a few sample files for you:

- a `foo_test.go` file under the test suite's root directory, including a `TestMain` method representing the test suite. This Go file will represent **the main program** to be executed in the tests, containing a boilerplate structure that includes:
    - a `FooTestSuite` Go struct representing the state to be passed across steps in the same scenario.
    - global variables to be passed across steps and scenarios, creating APM traces and spans for the life cycle methods of Godog (before/after Step/Scenario/Suite)
    - initialisation methods for `godog`'s life cycle hooks: _InitializeFooScenarios_ and _InitializeFooTestSuite_.
        - **InitializeFooTestSuite**: contains the life cycle hooks for the suite (`BeforeSuite and AfterSuite`)
        - **InitializeFooScenarios**: contains the life cycle hooks for each test scenario (`BeforeScenario, AfterScenario, BeforeStep and AfterStep`)
- a `docker-compose.yml` file under `internal/config/compose/profiles/foo`, for the runtime dependencies. This descriptor includes the definition of the services that are needed by our tests before they are run. The sample file contains an Elasticsearch instance, but it could include Kibana, Fleet Server, or any other service in the form of a Docker container.
- a `foo.feature` feature file under the **features** directory. This directory is the default location for the Gherkin feature files. Although it can be changed to any other location, using the `opts` structure, we recommend keeping it with the default value:

```go
    // changing the default path is not recommended
    opts := godog.Options{
		Paths:     []string{"new-location"},
	}
```

You are able now to read the content of the sample feature file - `vim features/foo.feature`

``` gherkin
Feature: eat godogs
  In order to be happy
  As a hungry gopher
  I need to be able to eat godogs

  Scenario: Eat 5 out of 12
    Given there are 12 godogs
    When I eat 5
    Then there should be 7 remaining
```

### Step 4 - Create godog step definitions

**NOTE:** same as **go test**, `godog` respects package level isolation. All your step definitions should be in your test suite root directory. In this case: **foo**.

From the test suite directory, you can directly run Go commands to run the tests: - `go test -v --godog.format=pretty`, or leverage the build system: - `make functional-test`

You should see that the steps are undefined:

```shell
Feature: Foo
  In order to be happy
  As a hungry gopher
  I need to be able to eat godogs

  Scenario: Eat 5 out of 12          # features/foo.feature:7
    Given there are 12 godogs
    When I eat 5
    Then there should be 7 remaining

1 scenarios (1 undefined)
3 steps (3 undefined)
1.131094831s

You can implement step definitions for undefined steps with these snippets:

func iEat(arg1 int) error {
        return godog.ErrPending
}

func thereAreGodogs(arg1 int) error {
        return godog.ErrPending
}

func thereShouldBeRemaining(arg1 int) error {
        return godog.ErrPending
}

func InitializeScenario(ctx *godog.ScenarioContext) {
        ctx.Step(`^I eat (\d+)$`, iEat)
        ctx.Step(`^there are (\d+) godogs$`, thereAreGodogs)
        ctx.Step(`^there should be (\d+) remaining$`, thereShouldBeRemaining)
}

testing: warning: no tests to run
PASS
ok      github.com/elastic/e2e-testing/e2e/_suites/foo  1.445s
```

Copy the content of the outputted `InitializeScenario` method, _the ctx.Step invocations_, into your `foo_test.go`, within the `InitializeFooScenario` struct - `vim foo_test.go`
``` go
func InitializeFooScenario(ctx *godog.ScenarioContext) {
    // ...
    ctx.Step(`^I eat (\d+)$`, iEat)
    ctx.Step(`^there are (\d+) godogs$`, thereAreGodogs)
    ctx.Step(`^there should be (\d+) remaining$`, thereShouldBeRemaining)
    // ...
}
```

Finally copy the outputted implementation methods in the `foo_test.go` file.

Run godog again - `go test -v --godog.format=pretty`

You should now see that the scenario is pending with one step pending and two steps skipped:
```
Feature: Foo
  In order to be happy
  As a hungry gopher
  I need to be able to eat godogs

  Scenario: Eat 5 out of 12          # features/foo.feature:7
    Given there are 12 godogs        # foo_test.go:112 -> github.com/elastic/e2e-testing/e2e/_suites/foo.thereAreGodogs
      TODO: write pending definition
    When I eat 5                     # foo_test.go:108 -> github.com/elastic/e2e-testing/e2e/_suites/foo.iEat
    Then there should be 7 remaining # foo_test.go:116 -> github.com/elastic/e2e-testing/e2e/_suites/foo.thereShouldBeRemaining

1 scenarios (1 pending)
3 steps (1 pending, 2 skipped)
1.025856125s
testing: warning: no tests to run
PASS
ok      github.com/elastic/e2e-testing/e2e/_suites/foo  1.331s
```

You may change **return godog.ErrPending** to **return nil** in the three step definitions and the scenario will pass successfully.

Also, you may omit error return if your step does not fail.

```go
func iEat(arg1 int) {
	// Eat arg1.
}
```

### Step 5 - Add some logic to the step definitions

At this time we recommend moving all the implementation methods to the `FooTestSuite` struct, so that we can pass state and logic between a scenario's steps with ease.


```go
func InitializeFooScenario(ctx *godog.ScenarioContext) {
    // ...
    ctx.Step(`^I eat (\d+)$`, testSuite.iEat)
    // ...
}

func (fts *FooTestSuite) iEat(arg1 int) {
	// Eat arg1.
}
```

### Step 6 - Add your custom logic to the step definitions

Now that you created your first scenario, you can continue with the next steps: working on an existing test suite, but it's probably a good idea to check out our brief introduction to `Test specification with Gherkin` below. It will provide you the bare minimal concepts about Gherkin clauses and BDD.

## Test specification with Gherkin

All the Gherkin (Cucumber) specifications are written in `.feature` files. The anatomy of a feature file is:

- **@tag_name**: A `@` character indicates a tag. And tags are used to filter the test execution. Tags could be placed on Features (applying the entire file), or on Scenarios (applying just to them). At this moment we are tagging each feature file with a tag using module's name, so that we can instrument the test runner to just run one. *more below.
- **Feature: File name**: Description of the group of uses cases (scenarios) in this feature file, in plain English. The feature file should contain just one of this keyword, and the name of the feature file must match the words used to name the Feature, as described in the official Gherkin lint: https://github.com/funkwerk/gherkin_lint/blob/master/features/file_name_differs_feature_name.feature.
>   _As a Business Analyst I want to be if file and feature names differ so that reader understand the feature just by the file name_
- **Scenario**: the name of a specific use case in plain English. The feature file can contain multiple scenarios.
- **Scenario Outline**: exactly the same as above, but we are are telling Cucumber that this use case has a matrix, so it has to iterate through an **Examples** table, interpolating those values into the placeholders in the scenario. The feature file can contain multiple outlined scenarios.
- **Given, Then, When, And, But keywords**: They represent a step in the scenario. Their order and meaning is extremely important in order to understand the use case they are part of, although they have no real impact in how we use them. If we use `doble quotes` around one or more words, ,`"version"`, that will tell Cucumber the presence of a fixed variable, with value the word/s among the double quotes. These variables will be the input parameters of the implementation functions in Go code. If we use `angles` around one or more words,`<version>`, that will tell Cucumber the presence of a dynamic variable, taken from the examples table. It's possible to combine both styles: `"<version>"`.
    - **Given** (Optional): It must tell an ocational reader what state must be in place for the use case to be valid.
    - **When** (Mandatory): It must tell an ocational reader what action or actions trigger the use case.
    - **Then** (Mandatory): It must tell an ocational reader what outcome has been generated after the use case happens.
    - **And** (Optional): Used within any of the above clauses, it must tell an ocational reader a secondary preparation (Given), trigger (When), or output (Then) that must be present.
    - **But** (Optional): Used within any of the above clauses, it must tell an ocational reader a secondary preparation (Given), trigger (When), or output (Then) that must not be present.
- **Examples:** (Mandatory with Scenario Outline): this `markdown table` will represent the elements to interpolate in the existing dynamic variables in the use case, being each column header the name of the different variables in the table. Besides that, each row will result in a test execution.

A good example could be [this one](./_suites/fleet/features/manage_integrations.feature).

## Working on existing test suite

We hope you enjoyed the `Test specification with Gherkin` introduction. If you find any gap, please no doubt in contributing it or opening an issue. Let's continue with our example, but this time let's work on an existing test suite.

Let's walk through a quick example to see how to start working with the e2e-testing framework, working on an existing test suite:

### Step 1 - Get familiar with the struct representing the test suite

It's usually named after the test suite, so you'll find it easy to find it. In that struct you'll find fields that are passed across steps in the same scenario, so pay attention on how they are used, and how they are reset after the scenario finishes.

It's also important to dedicate time to the life cycle hook methods: `BeforeStep` and `AfterStep`, `BeforeScenario` and `AfterScenario` and finally `BeforeSuite` and `AfterSuite`. They provide great information about the preparation and destruction of the runtime dependencies, or any other state that has be kept between steps or scenarios.

Ideally, no state should be passed among scenarios, as each of them should be idempotent, no matter the order it's executed.

### Step 2 - Get familiar with the internal package of the test framework

In this package we are defining helper code to accomplish certain repetitive tasks across the test steps. Tasks like querying a Kibana endpoint, or performing a search on an Elasticsearch instance, are good examples. For those two use cases we are providing clients to perform those operations. If your test code needs an operation that it's not supported there, feel free to add it to the clients' APIs, for future usage.

Another example for an internal API is the `internal/deploy` package, which contains the code to manage the life cycle of the services under tests using Docker containers or Kubernetes pods as unit of deployment.

Files under the `internal` package are usually unit tested, so please write a unit test when working on this package, or if you see the lack of tests in there.

### Step 3 - Get familiar with the APM intrumentation code for the test suite

We are instrumenting the test framework so that each step, scenario and suite send APM traces and spans to a Cloud instance we own. We also are instrumenting the internal methods in the framework so that we can find observe the project discovering bottlenecks or step scenarios that are slower than expected.

The test framework uses the Elastic APM Go agent to instrument the code, so please read more about how to send traces and spans in [this guide](https://www.elastic.co/guide/en/apm/agent/go/current/api.html).

We correlate `transactions` with test scenarios, and each step in the scenario is represented by a `span`. Each span is passing a Go context to the internal methods, so that the APM Go agent is able to start/end spans using the caller's context as parent. This way we can produce a chain of spans belonging to the scenario transaction.

An example of this instrumentation chain can be found here:

```go
// InternalCalculation performs an internal calculation
func InternalCalculation(ctx context.Context, args interface{}...) error {
	span, _ := apm.StartSpanOptions(ctx, "Doing an internal calculation", "module.entity.action", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
    span.Context.SetLabel("arguments", args) // create a label for this span
	defer span.End()

    // ... internal calculation
}
```

### Step 4 - Get familiar with the test suite files

Each test suite defines specific Go files where the logic is decoupled. Ideally, all those files should live under the `main` package of that test suite, although this is not a requirement. We put on these Go files code that belongs exclusively to the test suite, so it cannot be reused. Reusable code should live under the `internal` package of the test framework.

Group functionality for the test suite in separate Go files, so that its maintenance is much easier than in a single, huge file.

### Step 5 - Get familiar with test framework workdir

The test framework has a workdir where it performs certain tasks, such as cloning a git repository, keeping the state between different test runs, or store the configuration files for each test suite, among others.

This workdir is located under the user home: `~/CURRENT_USERNAME/.op` for Unix systems, and `C:\Users\CURRENT_USERNAME\.op` for Windows.

Related to the configuration files for each test suite, and the code to load those files into the test framework, they are located under the `internal/config` package. This package is responsible for loading the default configuration files for services and profiles, extracting the bundled configuration files into the test framework workdir. 
