# Formal verification of Metricbeat module

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
