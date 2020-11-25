# Observability Helm charts End-To-End tests

## Motivation

Our goal is for the Observability team to execute this automated e2e test suite while developing the Helm charts for APM Server, Filebeat and Metricbeat. The tests in this folder assert that the use cases (or scenarios) defined in the `features` directory are behaving as expected.

## How do the tests work?

At the topmost level, the test framework uses a BDD framework written in Go, where we set
the expected behavior of use cases in a feature file using Gherkin, and implementing the steps in Go code.
The provisioning of services is accomplished using [Kind (Kubernetes in Docker)](https://kind.sigs.k8s.io/https://kind.sigs.k8s.io/) and [Helm](https://helm.sh/) packages.

The tests will follow this general high-level approach:

1. Install runtime dependencies creating a Kind cluster using the locally installed `kind` binary, happening at before the test suite runs.
1. Execute BDD steps representing each scenario. Each step will return an Error if the behavior is not satisfied, marking the step and the scenario as failed, or will return `nil`.

### Diagnosing test failures

The first step in determining the exact failure is to try and reproduce the test run locally, ideally using the DEBUG log level to enhance the log output. Once you've done that, look at the output from the test run.

### Running the tests

1. Clone this repository, say into a folder named `e2e-testing`.

   ``` shell
   git clone git@github.com:elastic/e2e-testing.git
   ```

2. Configure the version of the tools you want to test (Optional).

This is an example of the optional configuration:

   ```shell
   # Depending on the versions used, 
   export HELM_VERSION="3.4.1"
   export HELM_CHART_VERSION="7.10.0"  # version of the Elastic's Observability Helm charts
   export HELM_KUBERNETES_VERSION="1.18.2" # version of the cluster to be passed to kind
   ```

3. Install dependencies.

   - Install Helm 3.4.1
   - Install Kind 0.8.1
   - Install Go: `https://golang.org/doc/install` _(The CI uses [GVM](https://github.com/andrewkroh/gvm))_
   - Install godog (from project's root directory): `make -C e2e install-godog`

4. Run the tests.

   If you want to run the tests in Developer mode, which means reusing bakend services between test runs, please set this environment variable first:

   ```shell
   # It won't tear down the backend services (k8s cluster) after a test suite.
   export DEVELOPER_MODE=true
   ```

   ```shell
   cd e2e/_suites/helm
   OP_LOG_LEVEL=DEBUG godog
   ```

   The tests will take a few minutes to run, spinning up the Kubernetes cluster, installing the helm charts, and performing the test steps outlined earlier.

   As the tests are running they will output the results in your terminal console. This will be quite verbose and you can ignore most of it until the tests finish. Then inspect at the output of the last play that ran and failed. On the contrary, you could use a different log level for the `OP_LOG_LEVEL` variable, being it possible to use `DEBUG`, `INFO (default)`, `WARN`, `ERROR`, `FATAL` as log levels.

### Tests fail because the product could not be configured or run correctly

This type of failure usually indicates that code for these tests itself needs to be changed.

See the sections below on how to run the tests locally.

### One or more scenarios fail

Check if the scenario has an annotation/tag supporting the test runner to filter the execution by that tag. Godog will run those scenarios. For more information about tags: https://github.com/cucumber/godog/#tags

   ```shell
   OP_LOG_LEVEL=DEBUG godog -t '@annotation'
   ```

Example:

   ```shell
   OP_LOG_LEVEL=DEBUG godog -t '@apm-server'
   ```

### Setup failures

Sometimes the tests could fail to configure or start the kubernetes cluster, etc. To determine why
this happened, look at your terminal log in DEBUG mode. make sure there is not another test cluster:

```shell
# Will remove existing test cluster
kind delete cluster --name helm-charts-test-suite
```

Note what you find and file a bug in the `elastic/e2e-testing` repository, requiring a fix to the helm suite to properly configure and start the product.

### I cannot move on

Please open an issue here: https://github.com/elastic/e2e-testing/issues/new
