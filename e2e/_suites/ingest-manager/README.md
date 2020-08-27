# Ingest Manager End-To-End tests

## Motivation

Our goal is for the Ingest Manager team to execute this automated e2e test suite while developing the product. The tests in this folder assert that the use cases (or scenarios) defined in the `features` directory are behaving as expected.

## How do the tests work?

At the topmost level, the test framework uses a BDD framework written in Go, where we set
the expected behavior of use cases in a feature file using Gherkin, and implementing the steps in Go code.
The provisining of services is accomplish using Docker Compose and the [testcontainers-go](https://github.com/testcontainers/testcontainers-go) library.

The tests will follow this general high-level approach:

1. Install runtime dependencies as Docker containers via Docker Compose, happening at before the test suite runs. These runtime dependencies are defined in a specific `profile` for Ingest Manager, in the form of a `docker-compose.yml` file.
1. Execute BDD steps representing each scenario. Each step will return an Error if the behavior is not satisfied, marking the step and the scenario as failed, or will return `nil`.

## Known Limitations

Because this framework uses Docker as the provisioning tool, all the services are based on Linux containers. That's why we consider this tool very suitable while developing the product, but would not cover the entire support matrix for the product: Linux, Windows, Mac, ARM, etc.

For Windows or other platform support, we should build Windows images and containers or, given the cross-platform nature of Golang, should add the building blocks in the test framework to run the code in the ephemeral CI workers for the underlaying platform.

### Diagnosing test failures

The first step in determining the exact failure is to try and reproduce the test run locally, ideally using the DEBUG log level to enhance the log output. Once you've done that, look at the output from the test run.

#### (For Mac) Docker is not able to save files in a temporary directory

It's important to configure `Docker for Mac` to allow it accessing the `/var/folders` directory, as this framework uses Mac's default temporary directory for storing tempoorary files.

To change it, please use Docker UI, go to `Preferences > Resources > File Sharing`, and add there `/var/folders` to the list of paths that can be mounted into Docker containers. For more information, please read https://docs.docker.com/docker-for-mac/#file-sharing.

### Running the tests

1. Clone this repository, say into a folder named `e2e-testing`.

   ``` shell
   git clone git@github.com:elastic/e2e-testing.git
   ```

2. Configure the version of the product you want to test (Optional).

This is an example of the optional configuration:

   ```shell
   # There should be a Docker image for the runtime dependencies (elasticsearch, kibana, package registry)
   export STACK_VERSION=7.9.0
   # (Fleet mode) This environment variable will use a fixed version of the Elastic agent binary, obtained from
   # https://artifacts-api.elastic.co/v1/search/7.9.0/elastic-agent
   export ELASTIC_AGENT_DOWNLOAD_URL="https://snapshots.elastic.co/7.9.0-81c43e7b/downloads/beats/elastic-agent/elastic-agent-7.9.0-linux-x86_64.tar.gz"
   # (Fleet mode) This environment variable will use the snapshots produced by Beats CI. If the above variable
   # is set, this variable will take no effect
   export ELASTIC_AGENT_USE_CI_SNAPSHOTS="true"
   # (Stand-Alone mode) This environment variable will use the its value as the Docker tag produced by Beats CI (Please look up here: https://container-library.elastic.co/r/observability-ci/elastic-agent). Here you have two examples for tags:
   export ELASTIC_AGENT_VERSION="pr-20356"
   # or
   export ELASTIC_AGENT_VERSION="78a762c76080aafa34c52386341b590dac24e2df"
   ```

3. Define the proper Docker images to be used in tests (Optional).

    Update the Docker compose files with the local version of the images you want to use.

    >TBD: There is an initiative to automate this process to build the Docker image for a PR (or the local workspace) before running the tests, so the image is ready.

4. Install dependencies.

   - Install Go: `https://golang.org/doc/install` _(The CI uses [GVM](https://github.com/andrewkroh/gvm))_
   - Install godog (from project's root directory): `make -C e2e install-godog`

5. Run the tests.

   If you want to run the tests in Developer mode, which means reusing bakend services between test runs, please set this environment variable first:

   ```shell
   # It won't tear down the backend services (ES, Kibana, Package Registry) or agent services after a test suite. 
   export DEVELOPER_MODE=true
   ```

   ```shell
   cd e2e/_suites/ingest-manager
   OP_LOG_LEVEL=DEBUG godog
   ```

   The tests will take a few minutes to run, spinning up a few Docker containers representing the various products in this framework and performing the test steps outlined earlier.

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
   OP_LOG_LEVEL=DEBUG godog -t '@stand_alone_mode'
   ```

### Setup failures

Sometimes the tests could fail to configure or start a product such as Metricbeat, Elasticsearch, etc. To determine why 
this happened, look at your terminal log in DEBUG mode. If a `docker-compose.yml` file is not present please execute this command:

```shell
## Will remove tool's existing default files and will update them with the bundled ones
rm -fr ~/.op/compose
```

Note what you find and file a bug in the `elastic/e2e-testing` repository, requiring a fix to the ingest-manager suite to properly configure and start the product.

### I cannot move on

Please open an issue here: https://github.com/elastic/e2e-testing/issues/new
