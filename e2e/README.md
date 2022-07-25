# E2E Testing framework

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

#### Helm charts
- `HELM_CHART_VERSION`. Set this environment variable to the proper version of the Helm charts to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L43
- `HELM_VERSION`. Set this environment variable to the proper version of Helm to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L44
- `KIND_VERSION`. Set this environment variable to the proper version of Kind (Kubernetes in Docker) to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L45
- `KUBERNETES_VERSION`. Set this environment variable to the proper version of Kubernetes to be used in the current execution. Default: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L46

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
