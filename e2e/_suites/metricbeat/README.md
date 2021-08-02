# Integrations End-To-End tests

## Motivation

Our goal is for the Integrations team to execute this automated e2e test suite while developing the product. The tests in this folder assert that the use cases (or scenarios) defined in the `features` directory are behaving as expected.

## How do the tests work?

At the topmost level, the test framework uses a BDD framework written in Go, where we set
the expected behavior of use cases in a feature file using Gherkin, and implementing the steps in Go code.
The provisining of services is accomplish using Docker Compose and the [testcontainers-go](https://github.com/testcontainers/testcontainers-go) library.

The tests will follow this general high-level approach:

1. Install runtime dependencies as Docker containers via Docker Compose, happening at before the test suite runs. These runtime dependencies are defined in a specific `profile` for Metricbeat, in the form of a `docker-compose.yml` file.
1. Execute BDD steps representing each scenario. Each step will return an Error if the behavior is not satisfied, marking the step and the scenario as failed, or will return `nil`.

### Running the tests

1. Clone this repository, say into a folder named `e2e-testing`.

   ``` shell
   git clone git@github.com:elastic/e2e-testing.git
   ```

2. Configure the version of the product you want to test (Optional).

This is an example of the optional configuration:

   ```shell
   # There should be a Docker image for the runtime dependencies (elasticsearch)
   export STACK_VERSION="8.0.0-SNAPSHOT"
   export BEAT_VERSION="8.0.0-SNAPSHOT"
   # or
   # This environment variable will use the snapshots produced by Beats CI
   export BEATS_USE_CI_SNAPSHOTS="true"
   export GITHUB_CHECK_SHA1="01234567890"
   ```

3. Define the proper Docker images to be used in tests (Optional).

    Update the Docker compose files with the local version of the images you want to use.

    >TBD: There is an initiative to automate this process to build the Docker image for a PR (or the local workspace) before running the tests, so the image is ready.

4. Install dependencies.

   - Install Go, using the language version defined in the `.go-version` file at the root directory. We recommend using [GVM](https://github.com/andrewkroh/gvm), same as done in the CI, which will allow you to install multiple versions of Go, setting the Go environment in consequence: `eval "$(gvm 1.15.9)"`
   - Install integrations `make -C cli sync-integrations`
   - Install godog (from project's root directory): `make -C e2e install-godog`

5. Run the tests.

   If you want to run the tests in Developer mode, which means reusing bakend services between test runs, please set this environment variable first:

   ```shell
   # It won't tear down the backend services (Elasticsearch) after a test suite.
   export DEVELOPER_MODE=true
   ```

   ```shell
   cd e2e/_suites/metricbeat
   OP_LOG_LEVEL=DEBUG go test -v
   ```

   Optionally, you can run only one of the feature files
   ```shell
   cd e2e/_suites/metricbeat
   OP_LOG_LEVEL=DEBUG go test -timeout 60m -v --godog.tags='@mysql'
   ```

### I cannot move on

Please open an issue here: https://github.com/elastic/e2e-testing/issues/new
