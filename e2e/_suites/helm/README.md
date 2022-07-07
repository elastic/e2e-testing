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

### Running the tests

1. Clone this repository, say into a folder named `e2e-testing`.

   ``` shell
   git clone git@github.com:elastic/e2e-testing.git
   ```

2. Configure the version of the tools you want to test (Optional).

This is an example of the optional configuration:

   ```shell
   # Depending on the versions used,
   export HELM_VERSION="3.9.0"        # Helm version: for Helm v2.x.x we have to initialise Tiller right after the k8s cluster
   export HELM_CHART_VERSION="7.17.3"  # version of the Elastic's Observability Helm charts
   export KUBERNETES_VERSION="1.23.4" # version of the cluster to be passed to kind
   ```

3. Install dependencies.

   - Install Helm 3.9.0
   - Install Kind 0.12.0
   - Install Go, using the language version defined in the `.go-version` file at the root directory. We recommend using [GVM](https://github.com/andrewkroh/gvm), same as done in the CI, which will allow you to install multiple versions of Go, setting the Go environment in consequence: `eval "$(gvm 1.15.9)"`
   - Godog and other test-related binaries will be installed in their supported versions when the project is first built, thanks to Go modules and Go build system.

4. Run the tests.

   If you want to run the tests in Developer mode, which means reusing bakend services between test runs, please set this environment variable first:

   ```shell
   # It won't tear down the backend services (k8s cluster) after a test suite.
   export DEVELOPER_MODE=true
   ```

   ```shell
   cd e2e/_suites/helm
   OP_LOG_LEVEL=DEBUG go test -v
   ```

   Optionally, you can run only one of the feature files
   ```shell
   cd e2e/_suites/helm
   OP_LOG_LEVEL=DEBUG go test -timeout 90m -v --godog.tags='@apm-server'
   ```

## Diagnosing test failures

### Setup failures

Sometimes the tests could fail to configure or start the kubernetes cluster, etc. To determine why
this happened, look at your terminal log in DEBUG/TRACE mode. make sure there is not another test cluster:

```shell
# Will remove existing test cluster
kind delete cluster --name helm-charts-test-suite
```

Note what you find and file a bug in the `elastic/e2e-testing` repository, requiring a fix to the helm suite to properly configure and start the product.

### I cannot move on

Please open an issue here: https://github.com/elastic/e2e-testing/issues/new
