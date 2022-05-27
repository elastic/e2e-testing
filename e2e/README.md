# E2E Testing framework

> If you want to start creating a new test suite, please read [the quickstart guide](./QUICKSTART.md), but don't forget to come back here to better understand the framework.

 The E2E test project uses `BDD` (Behavioral Driven Development), which means the tests are defined as test scenarios (or simply `scenarios`). A scenario is written in **plain English**, using business language hiding any implementation details. Therefore, the words _"clicks", "input", "browse",  "API call"_ are NOT allowed. And we do care about having well-expressed language in the feature files. Why? Because we want to hide the implementation details in the tests, and whoever is reading the feature files is able to understand the expected behavior of each scenario. And for us that's the key when talking about real E2E tests: to exercise user journeys (scenarios) instead of specific parts of the UI (graphical or API).

We want to make sure that the different test suites in this project are covering the main use cases for their core functionalities. So for that reason we are adding different test suites acting as [smoke tests](http://softwaretestingfundamentals.com/smoke-testing/) to verify that each test suite meets the specifications described here with certain grade of satisfaction.

>Smoke Testing, also known as “Build Verification Testing”, is a type of software testing that comprises of a non-exhaustive set of tests that aim at ensuring that the most important functions work. The result of this testing is used to decide if a build is stable enough to proceed with further testing.

Finally, the test framework must be responsible for creating and destroying the run-time dependencies, basically the Elastic Stack, and provide a minimal set of shared utilities that can be reused across different test suites, such as a Kibana client, an Elasticsearch client, a Service provider, etc.

## Statements about the test framework
- It uses the `Given/When/Then` pattern to define scenarios.
  - `Given` a state is present (optional)
  - `When` an action happens (mandatory) 
  - `Then` an outcome is expected (mandatory)
  - "And/But" clauses are allowed to provide continuation on the above ones. (Optional)
- Because it uses BDD, it's possible to combine existing scenario steps (the given-when-then clauses) forming new scenarios, so with no code it's possible to have new tests. For that reason, the steps must be atomic and decoupled (as any piece of software: low coupling and high cohesion).
- It does not use the GUI at all, so no Selenium, Cypress or any other test library having expectations on graphical components. It uses Go as the programming language and Cucumber to create the glue code between the code and the plain English scenarios. In almost two years, we have demonstrated that the APIs are not changing as fast as the GUI components.
- APIs not changing does not mean zero flakiness. Because there are so many moving pieces (stack versions), beats versions, elastic-agent versions, cloud machines, network access, etc... There could be situations where the tests fail, but they are rarely caused by test flakiness. Instead, they are usually caused by: 1) instability of the all-together system, 2) a real bug, 3) gherkin steps that are not consistent.
- The project usually checks for JSON responses, OS processes state, Elasticsearch queries responses (using the ES client), to name a few. Kibana is basically at the core of the tests, because we hit certain endpoints and wait for the responses.
  - Yes, the e2e tests are first citizen consumers of Kibana APIs, so they could be broken at the moment an API change on Kibana. We have explored the idea of implementing `Contract-Testing` with [pact.io](https://pact.io) (not implemented but in the wish list).
  - A PoC was submitted to [kibana](https://github.com/elastic/kibana/pull/80384) and to [this repo](https://github.com/elastic/e2e-testing/pull/339) demonstrating the benefits of `Contract-Testing`.

## Test Suites
The project has 3 test suites: Fleet, Observability Helm charts and K8s autodiscover.

- `Fleet`: The biggest one. It covers scenarios where the elastic-agent is installed in real VMs in different OSs and architectures. 85 scenarios in total. Feature files: https://github.com/elastic/e2e-testing/blob/main/e2e/_suites/fleet/features/
- `Helm Charts`: filebeat, metricbeat and apm-server. 3 scenarios in total. Feature files: https://github.com/elastic/e2e-testing/blob/main/e2e/_suites/helm/features/
- `k8s-autodiscover`: uses kind to deploy an elastic-agent, filebeat, metricbeat and heartbeat and collect certain metrics. 18 scenarios in total. Feature files: https://github.com/elastic/e2e-testing/blob/main/e2e/_suites/kubernetes-autodiscover/features/

- The target audience is matching the aforementioned suites, although any other team can come and add their suites.

## CI execution
On the CI, the framework creates a VM for the Stack (Elasticsearch, Kibana and Fleet Server) first, using the current valid SNAPSHOT. It's valid because we have validated in a pull-request that a given snapshot is not breaking the tests. This process is automated: everyday, we receive an automated bump of the different snapshots for all the supported maintenance branches (main, 8.1, 8.0 and 7.17 as of today).
  - In the fleet test suite, Fleet Server is part of the Stack. The scenarios are executed connecting the VMs where the elastic-agent is installed to the Stack machine.
  - A pull-request in the elastic-agent repo runs the Fleet scenarios using the CI snapshots for that PR for the agent, and the stack version in the target branch of the pull request for the beats in play (filebeat, heartbeat, metricbeat)
  - A pull-request in Beats repo runs the Fleet scenarios using the CI snapshots for that PR for the beats, and the stack version in the target branch of the pull request for the elastic-agent.

Besides that, it's possible to run the tests for a PR on Kibana: the docker image of kibana will be built (~1h) and then it will be used as part of the stack. https://github.com/elastic/e2e-testing/tree/main/e2e/_suites/fleet#running-against-a-kibana-pull-request-locally

An important consideration is that the local developer experience changed when we moved from using Docker containers to real VMs, so the framework is not currently targeted to be run locally, only using the CI (manually or via pull-request). We are currently working on improving that part.

The @elastic/observablt-robots team is maintaining the framework and, because of the current needs on the Fleet team, the test scenarios. On the other hand, our main focus as the Robots team is to only maintain the core functionalities of the e2e test project, and let the teams maintain their scenarios by themselves with our assistance.

## Contributors and maintenance
We have received contributions from multiple teams in different aspects of the e2e tests project, so we are ecstatic to receive them:

  - Julia Bardi and Nicolas Chaulet, from Fleet team, frontend engineers, have contributed a few scenarios for Fleet
  - Eric Davis, QA engineer, has helped in the definition of the scenarios for Fleet
  - Igor Guz, from Security team, QA engineer, has contributed scenarios for the security-related integrations, such as Endpoint, Linux and System
  - Christos Markou and Jaime Soriano have contributed to the k8s-autodiscover test suite, which is maintained by @elastic/obs-cloudnative-monitoring.
  - Julien Lind, from Fleet, has helped in defining the support matrix in terms of what OSs and architectures need to be run for Fleet test suite
  - Julien Mailleret, from Infra, has contributed to the Helm charts test suite.
  - Anderson Queiroz, from Elastic Agent, is currently working on the MacOS support for running the tests on real Apple machines.

## Tooling
Please check the specific document for tooling [here](./tooling.md).

## Regression testing
We have built the project and the CI job in a manner that it is possible to override different parameters about projects versions, so that we can set i.e. the version of the Elastic Stack to be used, or the version of the Elastic Agent. We have built and maintain branches to test the most recent versions of the stack, each release that comes out we maintain for a brief period and drop support for the oldest, while always keeping 'master' (8.0) and the 7.16 maintenance line, too:

- **7.13**: (for example): will use `7.13` alias for the Elastic Stack (including Fleet Server), Agent and Endpoint / Beats
- **7.16**: will use `7.16` alias for the all noted components, always being on the cusp of development, ahead of / newer than the .x release that came before it
- **master**: will use `8.0.0-SNAPSHOT` for the Elastic Stack and the Agent, representing the current development version of the different products under test.

With that in mind, the project supports setting these versions in environment variables, overriding the pre-branch default ones.

### Overriding Product versions
We are going to enumerate the variables that will affect the product versions used in the tests, per test suite:

>It's important to notice that the 7.9.x branch in **Fleet** test suite uses different source code for the communications with Kibana Fleet plugin, as API endpoints changed from 7.9 to 7.10, so there could be some combinations that are broken. See https://github.com/elastic/e2e-testing/pull/348 for further reference about these breaking changes.

> Related to this compatibility matrix too, it's also remarkable that Kibana **Fleet** plugin should not allow to enroll an agent with a version higher than kibana (See https://github.com/elastic/kibana/blob/fed9a4fddcc0087ee9eca6582a2a84e001890f08/x-pack/test/fleet_api_integration/apis/agents/enroll.ts#L99).

#### Fleet
- `BEAT_VERSION`. Set this environment variable to the proper version of the Beats to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/70b1d3ddaf39567aeb4c322054b93ad7ce53e825/.ci/Jenkinsfile#L44
- `ELASTIC_AGENT_VERSION`. Set this environment variable to the proper version of the Elastic Agent to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/70b1d3ddaf39567aeb4c322054b93ad7ce53e825/.ci/Jenkinsfile#L44
- `ELASTIC_AGENT_DOWNLOAD_URL`. Set this environment variable if you know the bucket URL for an Elastic Agent artifact generated by the CI, i.e. for a pull request. It will take precedence over the `BEAT_VERSION` variable. Default empty: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L35

#### Helm charts
- `HELM_CHART_VERSION`. Set this environment variable to the proper version of the Helm charts to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L43
- `HELM_VERSION`. Set this environment variable to the proper version of Helm to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L44
- `KIND_VERSION`. Set this environment variable to the proper version of Kind (Kubernetes in Docker) to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L45
- `KUBERNETES_VERSION`. Set this environment variable to the proper version of Kubernetes to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L46

#### Kubernetes autodiscover charts
- `BEAT_VERSION`. Set this environment variable to the proper version of the Beat to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/70b1d3ddaf39567aeb4c322054b93ad7ce53e825/.ci/Jenkinsfile#L44
- `ELASTIC_AGENT_VERSION`. Set this environment variable to the proper version of the Elastic Agent to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/70b1d3ddaf39567aeb4c322054b93ad7ce53e825/.ci/Jenkinsfile#L44
- `KIND_VERSION`. Set this environment variable to the proper version of Kind (Kubernetes in Docker) to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L45
- `KUBERNETES_VERSION`. Set this environment variable to the proper version of Kubernetes to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L46

### Environment variables affecting the build
The following environment variables affect how the tests are run in both the CI and a local machine.

- `ELASTIC_APM_ACTIVE`: Set this environment variable to `true` if you want to send instrumentation data to our CI clusters. When the tests are run in our CI, this variable will always be enabled. Default value: `false`.
- `ELASTIC_APM_ENVIRONMENT`: Set this environment variable to `ci` to send APM data to Elastic Cloud. Otherwise, the framework will spin up local APM Server and Kibana instances. For the CI, it will read credentials from Vault. Default value: `local`.
- `SKIP_PULL`: Set this environment variable to prevent the test suite to pull Docker images and/or external dependencies for all components. Default: `false`
- `BEATS_LOCAL_PATH`: Set this environment variable to the base path to your local clone of Beats if it's needed to use the binary snapshots produced by your local build instead of the official releases. The snapshots will be fetched from the `${BEATS_LOCAL_PATH}/${THE_BEAT}/build/distributions` local directory. This variable is intended to be used by Beats developers, when testing locally the artifacts generated its own build. Default: empty.
- `GITHUB_CHECK_SHA1`: Set this environment variable to the git commit in the right repository to use the binary snapshots produced by the CI instead of the official releases. The snapshots will be downloaded from a bucket in Google Cloud Storage. This variable is used by the repository, when testing the artifacts generated by the packaging job. Default: empty.
- `GITHUB_CHECK_REPO`: Set this environment variable to the name of the Github repository where the above git SHA commit lives. Default: elastic-agent.
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
git checkout main
# Run the tests for a specific branch
TAGS="fleet_mode" \
    TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE \
    BEAT_VERSION="7.10.1" \
    ELASTIC_AGENT_VERSION="7.10.1" \
    make -C e2e/_suites/fleet functional-test
```
Or running by feature file:
```shell
# Use the proper branch
git checkout main
FEATURES="fleet_mode.feature" \
    TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE \
    BEAT_VERSION="7.10.1" \
    ELASTIC_AGENT_VERSION="7.10.1" \
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

To do so:

1. Navigate to Jenkins: https://beats-ci.elastic.co/job/e2e-tests/job/e2e-testing-mbp/
2. Login as a user
3. Select the base branch for the test code: main (for 8.0.0-SNAPSHOT), 7.16, or any other maintenance branch.
4. In the left menu, click on `Buid with Parameters`.
5. In the input parameters form, set the stack version (for Fleet) using the specific variables for the test suite.
6. (Optional) Set the product version (Fleet or Helm charts) using the specific variables for the test suite if you want to consume a different artifact.
7. Click the `Build` button at the bottom of the parameters form.

Here you have a video reproducing the same steps:
![](./regression-testing.gif)

### Running tests for a Beats pull request
Because we trigger the E2E tests for each Beats PR that is packaged, it's possible to manually trigger it using CI user interface. To achieve it we must navigate to Jenkins and run the tests in the specific branch the original Beats PR is targeting.

>For further information about packaging Beats, please read [Beat's CI docs](https://github.com/elastic/beats/blob/1de27eed058dd074b58c71094c7678b3536251cb/README.md#ci).

To do so:

1. Navigate to Jenkins: https://beats-ci.elastic.co/job/e2e-tests/job/e2e-testing-mbp/
1. Login as a user
1. Select the base branch for the test code: 7.14, 7.15, 7.16 or main.
1. In the left menu, click on `Buid with Parameters`.
1. In the input parameters form, keep the Beat version (for Fleet) as is, to use each branch's default version.
1. In the input parameters form, keep the stack version (for Fleet) as is, to use each branch's default version.
1. In the input parameters form, set the `GITHUB_CHECK_NAME` to `E2E Tests`. This value will appear as the label for the Github check for the E2E tests.
1. In the input parameters form, set the `GITHUB_CHECK_SHA1` to the `SHA1` of the last commit in your pull request. This value will allow us to modify the mergeable status of that commit with the Github check. Besides that, it will set the specific directory in the GCP bucket to look up the CI binaries.
1. In the input parameters form, set the `GITHUB_CHECK_REPO` to `elastic-agent` or `beats`, depending where the aforementioned SHA1 belongs.
1. Click the `Build` button at the bottom of the parameters form.

## Noticing the test framework
To generate the notice files for this project:

1. Execute `make notice` to generate NOTICE.txt file.
