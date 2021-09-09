# End-2-End tests for Observability projects

If you don't want to deep dive into the theory about BDD and the internals of this test framework, please read [the quickstart guide](./QUICKSTART.md), but don't forget to come back here to better understand the framework.

## Main goals

We want to make sure that the different test suites in this project are covering the main use cases for their core functionalities. So for that reason we are adding different test suites acting as [smoke tests](http://softwaretestingfundamentals.com/smoke-testing/) to verify that each test suite meets the specifications described here with certain grade of satisfaction.

>Smoke Testing, also known as “Build Verification Testing”, is a type of software testing that comprises of a non-exhaustive set of tests that aim at ensuring that the most important functions work. The result of this testing is used to decide if a build is stable enough to proceed with further testing.

Finally, the test framework must be responsible for creating and destroying the run-time dependencies, basically the Elastic Stack, and provide a minimal set of shared utilities that can be reused across different test suites, such as a Kibana client, an Elasticsearch client, a Service provider, etc.

## Tooling

As we want to run _E2E tests_, we need a manner to describe the functionality to implement in a _functional manner_. And it would be great if we are able to use plain English to specify how our software behaves, instead of using code. And if it's possible to automate the execution of that specification, even better.

The most accepted manner to achieve this executable specification in the software industry, using a high level approach that anybody in the team could understand and backed by a testing framework, is [`Cucumber`](https://cucumber.io). So we will use `Cucumber` to set the behaviours (use cases) of our software.

Then we need a manner to connect that plain English feature specification with code. Fortunately, `Cucumber` has a wide number of implementations (Java, Ruby, NodeJS, Go...), so we can choose one of them to implement our tests.

We are going to use Go for writing the End-2-End tests, so we would need the Go implementation for `Cucumber`. That implementation is [`Godog`](https://github.com/cucumber/godog), which is the glue between the spec files and the Go code. `Godog` is a wrapper over the `go test` command, so they are almost interchangeable when running the tests.

> In this test framework, we are running Godog with `go test`, as explained [here](https://github.com/cucumber/godog#running-godog-with-go-test).

### Cucumber: BDD at its core

The specification of these E2E tests has been done using `BDD` (Behaviour-Driven Development) principles, where:

>BDD aims to narrow the communication gaps between team members, foster better understanding of the customer and promote continuous communication with real world examples.

From Cucumber's website:

>Cucumber is a tool that supports Behaviour-Driven Development(BDD), and it reads executable specifications written in plain text and validates that the software does what those specifications say. The specifications consists of multiple examples, or scenarios.

The way we are going to specify our software is using [`Gherkin`](https://cucumber.io/docs/gherkin/reference/).

>Gherkin uses a set of special keywords to give structure and meaning to executable specifications. Each keyword is translated to many spoken languages. Most lines in a Gherkin document start with one of the keywords.

The key part here is **executable specifications**: we will be able to automate the verification of the specifications and potentially get a coverage of these specs.

### Godog: Cucumber for Go

From Godog's website:

>Package godog is the official Cucumber BDD framework for Go, it merges specification and test documentation into one cohesive whole.

For this test framework, we have chosen Godog over any other test framework because the Beats team (the team we started working with) is already using Go, so it seems reasonable to choose it.

## Technology stack

### Build system
The test framework makes use of `Make` to prepare the environment to run the `Cucumber` tests. Although it's still possible using the `godog` binary, or the idiomatic `go test`, to run the test suites and scenarios, we recommend using the proper `Make` goals, in particular the [`functional-test` goal](https://github.com/elastic/e2e-testing/blob/05cdf195dfae0cb886d30b1cf6a1ccd95ddba9f5/e2e/commons-test.mk#L56). We also provide [a set of example goals](https://github.com/elastic/e2e-testing/blob/05cdf195dfae0cb886d30b1cf6a1ccd95ddba9f5/e2e/Makefile#L22-L35) with different use cases for running most common scenarios.

Each test suite, which lives under the `e2e/_suites` directory, has it's own Makefile to control the build life cycle of the test project.

> It's possible to create a new test suite with `SUITE=name make -C e2e create-suite`, which creates the build files and the scaffolding for the first test suite.

### Services as Docker containers
The services provided by the test suites in this framework will be started in the form of Docker containers. To manage the life cycle of those containers in test time we are going to use [`Testcontainers`](https://golang.testcontainers.org), a set of libraries to simplify the usage of the Docker client, attaching container life cycles to the tests, so whenever the tests finish, the containers will stop in consequence.

### Runtime dependencies
In many cases, we want to store the metrics in Elasticsearch, so at some point we must start up an Elasticsearch instance. Besides that, we want to query the Elasticsearch to perform assertions on the metrics, such as there are no errors, or the field `f.foo` takes the value `bar`. For that reason we need an Elasticsearch instance in a well-known location. We are going to group this Elasticsearch instance, and any other runtime dependencies, under the concept of a `profile`, which is a represented by a `docker-compose.yml` file under the `cli/config/compose/profiles/` and the name of the test suite.

As an example, the Fleet test suite will need an Elasticsearch instance, Kibana and Fleet Server.

#### Configuration files
If the profile needs certain configuration files, we recommend locating them under a `configurations` folder in the profile directory. As an example, see `kibana.config.yml` in the `fleet` profile.

### Feature files
We will create use cases for the module in a separate `.feature` file, ideally named after the name of the feature to test (i.e. _apache.feature_), and located under the `features` directory of each test suite. These feature files are considered as requirement for Cucumber, and they will be parsed by the Godog test runner and matched against the Go code implementing the tests.

#### Feature files and the CI
There is [a descriptor file for the CI](../.ci/.e2e-tests.yaml) in which we define the parallel branches that will be created in the execution of a job. This YAML file defines suites and tags. A suite represents each test suite directory under the `e2e/_suites` directory, and the tags represent the tags will be passed to the test runner to filter the test execution. Another configuration we define in this file is related to the capabilities to run certain tags at the pull request stage, using the `pullRequestFilter` child element. This element will be appended to the tags used to filter the test runner.

Adding a new feature file will require to check [the aforementioned descriptor file](../.ci/.e2e-tests.yaml). If the tags in the new file are not there, you should add a new parallel branch under the main test suite, or update the tags to add the new scenarios in an existing parallel branch.

## Known Limitations
Because this framework uses Docker as the provisioning tool, all the services are based on Linux containers. That's why we consider this tool very suitable while developing the product, but would not cover the entire support matrix for the product: Linux, Windows, Mac, ARM, etc.

For Windows or other platform support, we are providing support to run the tests in the ephemeral CI workers for the underlaying platform: in other words, we are going to install the platform-specific binaries under test in a CI worker, connecting to the runtime dependencies of the test suite in a remote location (another worker, Elastic Cloud, etc.).

## Generating documentation about the specifications
If you want to transform your feature files into a nicer representation using HTML, please run this command from the root `e2e` directory to build a website for all test suites:

```shell
$ make build-docs
```

It will generate the website under the `./docs` directory (which is ignored in Git). You'll be able to navigate through any feature file and test scenario in a website.

## Regression testing
We have built the project and the CI job in a manner that it is possible to override different parameters about projects versions, so that we can set i.e. the version of the Elastic Stack to be used, or the version of the Elastic Agent. We have built and maintain branches to test the most recent versions of the stack, each release that comes out we maintain for a brief period and drop support for the oldest, while always keeping 'master' (8.0) and the 7.x maintainenace line, too:

- **7.13.x**: (for example): will use `7.13.x` alias for the Elastic Stack (including Fleet Server), Agent and Endpoint / Beats
- **7.x**: will use `7.x` alias for the all noted components, always being on the cusp of development, ahead of / newer than the .x release that came before it
- **master**: will use `8.0.0-SNAPSHOT` for the Elastic Stack and the Agent, representing the current development version of the different products under test.

With that in mind, the project supports setting these versions in environment variables, overriding the pre-branch default ones.

### Overriding Product versions
We are going to enumerate the variables that will affect the product versions used in the tests, per test suite:

>It's important to notice that the 7.9.x branch in **Fleet** test suite uses different source code for the communications with Kibana Fleet plugin, as API endpoints changed from 7.9 to 7.10, so there could be some combinations that are broken. See https://github.com/elastic/e2e-testing/pull/348 for further reference about these breaking changes.

> Related to this compatibility matrix too, it's also remarkable that Kibana **Fleet** plugin should not allow to enroll an agent with a version higher than kibana (See https://github.com/elastic/kibana/blob/fed9a4fddcc0087ee9eca6582a2a84e001890f08/x-pack/test/fleet_api_integration/apis/agents/enroll.ts#L99).

#### Fleet
- `BEAT_VERSION`. Set this environment variable to the proper version of the Elastic Agent to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/70b1d3ddaf39567aeb4c322054b93ad7ce53e825/.ci/Jenkinsfile#L44
- `ELASTIC_AGENT_DOWNLOAD_URL`. Set this environment variable if you know the bucket URL for an Elastic Agent artifact generated by the CI, i.e. for a pull request. It will take precedence over the `BEAT_VERSION` variable. Default empty: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L35
- `ELASTIC_AGENT_STALE_VERSION`. Set this environment variable to the proper version of the Elastic Agent to be used in the upgrade tests, representing the version to be upgraded. Default: See https://github.com/elastic/e2e-testing/blob/b8d0cb09d575f90f447fe3331b6df0a185c01c89/.ci/Jenkinsfile#L38

#### Helm charts
- `HELM_CHART_VERSION`. Set this environment variable to the proper version of the Helm charts to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L43
- `HELM_VERSION`. Set this environment variable to the proper version of Helm to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L44
- `KIND_VERSION`. Set this environment variable to the proper version of Kind (Kubernetes in Docker) to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L45
- `KUBERNETES_VERSION`. Set this environment variable to the proper version of Kubernetes to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L46

#### Kubernetes autodiscover charts
- `BEAT_VERSION`. Set this environment variable to the proper version of the Beat to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/70b1d3ddaf39567aeb4c322054b93ad7ce53e825/.ci/Jenkinsfile#L44
- `KIND_VERSION`. Set this environment variable to the proper version of Kind (Kubernetes in Docker) to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L45
- `KUBERNETES_VERSION`. Set this environment variable to the proper version of Kubernetes to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L46

### Environment variables affecting the build
The following environment variables affect how the tests are run in both the CI and a local machine.

- `ELASTIC_APM_ACTIVE`: Set this environment variable to `true` if you want to send instrumentation data to our CI clusters. When the tests are run in our CI, this variable will always be enabled. Default value: `false`.
- `ELASTIC_APM_ENVIRONMENT`: Set this environment variable to `ci` to send APM data to Elastic Cloud. Otherwise, the framework will spin up local APM Server and Kibana instances. For the CI, it will read credentials from Vault. Default value: `local`.
- `SKIP_PULL`: Set this environment variable to prevent the test suite to pull Docker images and/or external dependencies for all components. Default: `false` 
- `BEATS_LOCAL_PATH`: Set this environment variable to the base path to your local clone of Beats if it's needed to use the binary snapshots produced by your local build instead of the official releases. The snapshots will be fetched from the `${BEATS_LOCAL_PATH}/${THE_BEAT}/build/distributions` local directory. This variable is intended to be used by Beats developers, when testing locally the artifacts generated its own build. Default: empty.
- `BEATS_USE_CI_SNAPSHOTS`: Set this environment variable to `true` if it's needed to use the binary snapshots produced by Beats CI instead of the official releases. The snapshots will be downloaded from a bucket in Google Cloud Storage. This variable is used by the Beats repository, when testing the artifacts generated by the packaging job. Default: `false`.
- `LOG_LEVEL`: Set this environment variable to `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR` or `FATAL` to set the log level in the project. Default: `INFO`.
- `DEVELOPER_MODE`: Set this environment variable to `true` to activate developer mode, which means not destroying the services provisioned by the test framework. Default: `false`.
- `KIBANA_VERSION`. Set this environment variable to the proper version of the Kibana instance to be used in the current execution, which should be used for the Docker tag of the kibana instance. It will refer to an image related to a Kibana PR, under the Observability-CI namespace. Default is empty 
- `STACK_VERSION`. Set this environment variable to the proper version of the Elasticsearch to be used in the current execution. The default value depens on the branch you are targeting your work.
    - **master (Fleet):** https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/e2e/_suites/fleet/ingest-manager_test.go#L39
- `TIMEOUT_FACTOR`: Set this environment variable to an integer number, which represents the factor to be used while waiting for resources within the tests. I.e. waiting for Kibana needs around 30 seconds. Instead of hardcoding 30 seconds, or 3 minutes, in the code, we use a backoff strategy to wait until an amount of time, specific per situation, multiplying it by the timeout factor. With that in mind, we are able to set a higher factor on CI without changing the code, and the developer is able to locally set specific conditions when running the tests on slower machines. Default: `3`.

- `FEATURES`: Set this environment variable to an existing feature file, or a glob expression (`fleet_*.feature`), that will be passed to the test runner to filter the execution, selecting those feature files matching that expression. If empty, all feature files in the `features/` directory will be used. It can be used in combination with `TAGS`.
- `TAGS`: Set this environment variable to [a Cucumber tag expression](https://github.com/cucumber/godog#tags), that will be passed to the test runner to filter the execution, selecting those scenarios matching that expresion, across any feature file. It can be used in combination with `FEATURES`.
- `SKIP_SCENARIOS`: Set this environment variable to `false` if it's needed to include the scenarios annotated as `@skip` in the current test execution, adding that taf to the `TAGS` variable. Default value: `true`.

### Running regressions locally

The tests will take a few minutes to run, spinning up a few Docker containers (or Kubernetes pods) representing the various runtime dependencies for the test suite and performing the test steps outlined earlier.

As the tests are running they will output the results in your terminal console. This will be quite verbose and you can ignore most of it until the tests finish. Then inspect at the output of the last play that ran and failed. On the contrary, you could use a different log level for the `OP_LOG_LEVEL` variable, being it possible to use `DEBUG`, `INFO (default)`, `WARN`, `ERROR`, `FATAL` as log levels.

In the following example, we will run the Fleet tests for the 8.0.0-SNAPSHOT stack with the released 7.10.1 version of the agent.

```shell
# Use the proper branch
git checkout master
# Run the tests for a specific branch
TAGS="fleet_mode_agent" \
    TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE \
    BEAT_VERSION="7.10.1" \
    make -C e2e/_suites/fleet functional-test
```
Or running by feature file:
```shell
# Use the proper branch
git checkout master
FEATURES="fleet_mode_agent.feature" \
    TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE \
    BEAT_VERSION="7.10.1" \
    make -C e2e/_suites/fleet functional-test
```

When running regression testing locally, please make sure you clean up tool's workspace among runs.

```shell
# It will remove $HOME/.op/compose files
make clean-workspace
```

If you want to refresh the Docker images used by the tests:

```shell
# It will remove and pull the images used in the current branch. Breathe, it will take time.
make clean-docker
```

>`make clean` will do both clean-up operations

### Running regressions on CI
Because we are able to parameterize a CI job, it's possible to run regression testing with different versions of the stack and the products under test. To achieve it we must navigate to Jenkins and run the tests with different combinations for each product.

> Note, as of this PR: https://github.com/elastic/e2e-testing/pull/669/files we have implemented a mechanism to NOT run tests marked as `@nightly` during PR CI test runs, if they, for any reason, are not capable of successfully finishing for a given reason.  The foremost example is the Agent upgrade tests which do not run on PR CI due to the lack of proper signing for the binaries needed.  The tag is implemented basically as a "nightly only" citation.

To do so:

1. Navigate to Jenkins: https://beats-ci.elastic.co/job/e2e-tests/job/e2e-testing-mbp/
1. Login as a user
1. Select the base branch for the test code: master (for 8.0.0-SNAPSHOT), 7.x, or any other maintenance branch.
1. In the left menu, click on `Buid with Parameters`.
1. In the input parameters form, set the stack version (for Fleet) using the specific variables for the test suite.
1. (Optional) Set the product version (Fleet or Helm charts) using the specific variables for the test suite if you want to consume a different artifact.
1. Click the `Build` button at the bottom of the parameters form.

Here you have a video reproducing the same steps:
![](./regression-testing.gif)

### Running tests for a Beats pull request
Because we trigger the E2E tests for each Beats PR that is packaged, it's possible to manually trigger it using CI user interface. To achieve it we must navigate to Jenkins and run the tests in the specific branch the original Beats PR is targeting.

>For further information about packaging Beats, please read [Beat's CI docs](https://github.com/elastic/beats/blob/1de27eed058dd074b58c71094c7678b3536251cb/README.md#ci).

To do so:

1. Navigate to Jenkins: https://beats-ci.elastic.co/job/e2e-tests/job/e2e-testing-mbp/
1. Login as a user
1. Select the base branch for the test code: 7.10.x, 7.11.x, 7.x or master.
1. In the left menu, click on `Buid with Parameters`.
1. In the input parameters form, keep the Beat version (for Fleet) as is, to use each branch's default version.
1. In the input parameters form, keep the stack version (for Fleet) as is, to use each branch's default version.
1. In the input parameters form, set the `GITHUB_CHECK_NAME` to `E2E Tests`. This value will appear as the label for the Github check for the E2E tests.
1. In the input parameters form, set the `GITHUB_CHECK_REPO` to `beats`.
1. In the input parameters form, check the `BEATS_USE_CI_SNAPSHOTS` checkbox. This value will instrument the test framework to download the binaries from the CI bucket on Google Cloud Platform Storage.
1. In the input parameters form, set the `GITHUB_CHECK_SHA1` to the `SHA1` of the last commit in your pull request. This value will allow us to modify the mergeable status of that commit with the Github check. Besides that, it will set the specific directory in the GCP bucket to look up the CI binaries.
1. Click the `Build` button at the bottom of the parameters form.

## Noticing the test framework

To generate the notice files for this project:

1. Execute `make notice` to generate NOTICE.txt file.
