# Fleet End-To-End tests

## Motivation

Our goal is for the Fleet team to execute this automated e2e test suite while developing the product. The tests in this folder assert that the use cases (or scenarios) defined in the `features` directory are behaving as expected.

## How do the tests work?

At the topmost level, the test framework uses a BDD framework written in Go, where we set
the expected behavior of use cases in a feature file using Gherkin, and implementing the steps in Go code.
The provisining of services is accomplish using Docker Compose and the [testcontainers-go](https://github.com/testcontainers/testcontainers-go) library.

The tests will follow this general high-level approach:

1. Install runtime dependencies as Docker containers via Docker Compose, happening at before the test suite runs. These runtime dependencies are defined in a specific `profile` for Fleet, in the form of a `docker-compose.yml` file.
1. Execute BDD steps representing each scenario. Each step will return an Error if the behavior is not satisfied, marking the step and the scenario as failed, or will return `nil`.

## Running against remote Docker

This framework supports running tests against a remote docker daemon. To enable this feature a passwordless ssh key is required for unattended test runs. To run the test against a remote docker the environment variable **DOCKER_HOST** should be set, for example:

```shell
DOCKER_HOST="ssh://user@192.168.1.15"
```

This will tell the test framework to connect to the remote docker daemon over ssh and will also correctly set the base urls for accessing Kibana and Elasticsearch api endpoints from your local machine.

You may be able to speed up tests run this way by altering some ssh settings in **~/.ssh/config** on your local machine:

```
Host 192.168.1.15 # replace with your remote host
  controlmaster yes
  controlpath ~/.ssh/sockets/%r@%h-%p
  controlpersist yes
```

- Note that **~/.ssh/sockets** directory must already exist.
- Note that docker uses an incredibly large number of ssh connections this way, it may require increasing the max open files on the remote host (Linux). To do so edit **/etc/security/limits.conf** and append the following:

```
* - nofile 500000
```

To verify this took place, logout and back in and run `ulimit -n`

```
$ ulimit -n
500000
```

## Running against a remote deployed stack

If an existing Elasticsearch, Kibana, Fleet server is already up and running, you can run the e2e tests against that existing cluster. The following environment variables are required:

```
PROVIDER=remote
```

We set the provider to manual, meaning there is no bootstrapping or deploying of required services as it is expected that those requirements be met prior to running the tests. Next, we need to point our tests to the service endpoints in order to perform the necessary operations against the Fleet server:

```
KIBANA_URL=https://a.public.ip:a.public.port
ELASTICSEARCH_URL=https://a.public.ip:a.public.port
FLEET_URL=https://a.public.ip:a.public.port
```

The above variables need to be accessible by the tests, if running the stack behind a firewall, ports may need to be exposed manually. The usage of `http` vs `https` is not important as our tests primarily deal with self signed certficates that are not validated against a true certficate authority.

### Running the tests

1. Clone this repository, say into a folder named `e2e-testing`.

   ``` shell
   git clone git@github.com:elastic/e2e-testing.git
   ```

2. Configure the version of the product you want to test (Optional).

This is an example of the optional configuration:

   ```shell
   # There should be a Docker image for the runtime dependencies (elasticsearch, package registry)
   export STACK_VERSION=8.0.0-SNAPSHOT
   # There should be a Docker image for the runtime dependencies (kibana)
   export KIBANA_VERSION=pr12345
   # (Fleet mode) This environment variable will use a fixed version of the Elastic agent binary, obtained from
   # https://artifacts-api.elastic.co/v1/search/8.0.0-SNAPSHOT/elastic-agent
   export ELASTIC_AGENT_DOWNLOAD_URL="https://snapshots.elastic.co/8.0.0-59098054/downloads/beats/elastic-agent/elastic-agent-8.0.0-SNAPSHOT-linux-x86_64.tar.gz"
   # This environment variable will use the its value as the Docker tag produced by Beats CI (Please look up Google Cloud Storage CI bucket).
   export GITHUB_CHECK_SHA1="78a762c76080aafa34c52386341b590dac24e2df"
   ```

3. Define the proper Docker images to be used in tests (Optional).

    Update the Docker compose files with the local version of the images you want to use.

    >TBD: There is an initiative to automate this process to build the Docker image for a PR (or the local workspace) before running the tests, so the image is ready.

4. Install dependencies.

   - Install Go, using the language version defined in the `.go-version` file at the root directory. We recommend using [GVM](https://github.com/andrewkroh/gvm), same as done in the CI, which will allow you to install multiple versions of Go, setting the Go environment in consequence: `eval "$(gvm 1.15.9)"`
   - Godog and other test-related binaries will be installed in their supported versions when the project is first built, thanks to Go modules and Go build system.

5. Run the tests.

   If you want to run the tests in Developer mode, which means reusing bakend services between test runs, please set this environment variable first:

   ```shell
   # It won't tear down the backend services (ES, Kibana, Package Registry) or agent services after a test suite.
   export DEVELOPER_MODE=true
   ```

   ```shell
   cd e2e/_suites/fleet
   OP_LOG_LEVEL=DEBUG go test -v
   ```

   Optionally, you can run only one of the feature files
   ```shell
   cd e2e/_suites/fleet
   OP_LOG_LEVEL=DEBUG go test -timeout 60m -v --godog.tags='@fleet_mode_agent'
   ```

### Running Kibana with different configuration file
In the case you need to run Kibana with a different set of properties, it's possible to do so simply using the `kibana uses "my-custom-profile" profile` step. This step, if executed at the very beginning of an scenario, or as a `Background` step for all scenarios, will execute the _Bootstrap_ code with the configuration located at `my-custom-profile`. As a reminder, `Bootstrap` will reevaluate the state of the runtime dependencies, recreating those that changed.

In order to achieve that you have to:

1. create a `kibana.config.yml` at `$E2E_ROOT_DIR/cli/config/compose/profiles/fleet/my-custom-profile/` file with your own properties. _You need to commit this file to the repository_.
2. add the `kibana uses "my-custom-profile" profile` step in any of the following cases:
   a. for an entire test suite adding a `Background` step like this:
```gherkin
Background: Setting up kibana instance with my custom profile
  Given kibana uses "my-custom-profile" profile
```

   b. for a single test scenario adding a `Given` clause at the beginning. In this case, make sure you set the `default` profile as a `Background` so that it restores the Kibana state at the beginning of the next scenario.
```gherkin
Background: Setting up kibana instance with default profile
  Given kibana uses "default" profile
```
3. Run the tests! Kibana will be recreated with the profile configuration in those scenarios using the new step.

### Fleet UI e2e tests CI job

https://beats-ci.elastic.co/job/e2e-tests/job/e2e-testing-kibana-fleet/build?delay=0sec

### Running against a Kibana pull request locally

1. Build kibana docker image from pull request using this job: (Custom Kibana - Deploy)[https://apm-ci.elastic.co/job/apm-shared/job/oblt-test-env/job/custom-kibana-deploy/build?delay=0sec]
   - Provide `kibana_branch` parameter to refer to your pr number e.g. `PR/100000`
   - Skip deploy_kibana step
2. Set envvar to pr
`export KIBANA_VERSION=pr100000`
3. Run tests

### Running against a Kibana running locally

1. Set envvars
```
export PROVIDER=remote
export KIBANA_URL=http://localhost:5601
export ELASTICSEARCH_URL=http://localhost:9200
export FLEET_URL=http://localhost:8220
```
2. Run tests

### Need help?

Please open an issue here: https://github.com/elastic/e2e-testing/issues/new
