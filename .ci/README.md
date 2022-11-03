# CI considerations

## Jenkins Stages and platform support
There are a set of YAML files, named after `.e2e-tests*.yaml`, that drive the execution of the different test scenarios and suites in the CI. These files are read by Jenkins at build time creating as many Jenkins parallel branches as items declared in the files. Ideally, each parallel branch should run a set of tests in a specific platform, i.e. the _"apm-server scenarios on Centos 8"_.

### Available platforms file structure
To support multiple test suite descriptors, we are defining the available platforms in one single file: `.e2e-platforms.yaml`. The structure of the file is the following:

- **PLATFORMS**: this entry will hold a YAML object with all the available platforms where the tests could be run. Each available platform will be defined as a key in the YAML object, where each object will have a key identifying the platform with a very descriptive name (i.e. `stack` or `debian_arm64`) and the following attributes:
  - **description**: Description of the purpose and/or characteristics of the platform machine. Required.
  - **image**: the AWS AMI identifier, i.e. `ami-0d90bed76900e679a`. Required.
  - **instance_type**: the AWS instance type, representing the size of the machine, i.e. `c5.4xlarge`. Required.
  - **username**: the default user name when connecting to the machine using SSH, used by Ansible to execute commands on the remote machine. I.e. `centos`, `admin` or `ec2-user`. Required.

In order to configure each platform, there is an `Ansible` script that installs the runtime dependencies for the machine, such as specific OS packages or libraries. Any available platform must declare its own block in the [`./ansible/tasks/install_deps.yml`](./ansible/tasks/install_deps.yml) file in order to be considered as supported.

### Test suites and scenarios file structure
It's possible that a consumer of the e2e tests would need to define a specific layout for the test execution, adding or removing suites and/or scenarios. That's the case for Beats or the Elastic Agent, which triggers the E2E tests with a different layout than for the own development of the test framework: while in Beats or the Elastic Agent we are more interested in running the test for Fleet only, when developing the project we want to verify all the test suites at a time. The structure of these files is the following:

- **SUITES**: this entry will hold a YAML object containing a list of suite. Each suite in the list will be represented by a YAML object with the following attributes:
  - **suite**: the name of the suite. Will be used to look up the root directory of the test suite, located under the `e2e/_suites` directory. Therefore, only `fleet`, `helm` and `kubernetes-autodiscover` are valid values. Required.
  - **provider**: declares the provider type for the test suite. Valid values are `docker`, `elastic-package` and `remote`. If not present, it will use `remote` as fallback. Optional.
  - **scenarios**: a list of YAML objects representing the test scenarios, where the tests are executed. A test scenario will basically declare how to run a set of test, using the following attributes:
    - **name**: name of the test scenario. It will be used by Jenkins to name the parallel stage representing this scenario. Required.
    - **provider**: declares the provider type for the test scenario. Valid values are `docker`, `elastic-package` and `remote`. If not present, it will use its parent test suite's provider. Optional.
    - **tags**: a Gherkin expression to filter scenarios by tag. It will drive the real execution of the tests, selecting which feature files and/or Cucumber tags will be added to the current test execution. An example could be `linux_integration` or `running_on_beats`. For reference, see https://github.com/cucumber/godog#tags. Required.
    - **platforms**: a list of platforms where the tests will be executed. Valid values are already declared under the `PLATFORMS` object, using the key of the platform as elements in the list. I.e. `["centos8_arm64", "centos8_amd64", "debian_arm64", "debian_amd64", "sles15", "ubuntu22", "windows2019"]`. Required.

### Adding a new supported platform

You could be insterested in adding a new operative system in a specific architecture (AMD/ARM)

1. Look up AWS community AMIs in the Ohio (`us-east-2`) region. Take a note on its AMI ID and the default use, as you'll need them later on.
2. Add a new entry in the `.e2e-platforms.yaml` file, following the same structure, as described above. The default user of the instance is very important, as it's used to log in in the remote machine once created by the tests.
3. Select an AWS machine type for the new platform, using the existing platforms file as a reference. Our team is actively checking how much cloud resources are consumed by the tests, so please consider using a machine type that already exists.
 
> If you need a machine type that is bigger or powerful, please open a discussion with us so we can understand your needs.

4. Make sure you install the required software dependencies using Ansible. Please take a look at [this file](./ansible/tasks/install_deps.yml). You will find in it the package manager commands and specific filters for the different OS families that are supported by Ansible.

> It's very likely that the new platform is already covered, as there are tasks for Debian/Ubuntu, CentOS (Fedora, RedHat), and Oracle Linux.

5. Add the new supported platform to the test execution, as described above: open all the test descriptors (i.e. `./.e2e-tests-*.yaml`) and add your new platform to the scenarios you are interested in, as a new platform in the `platforms` array.

## Running a CI Deployment

### Prereqs

- In order to interact with the build system, i.e. running Ansible commands to provision the AWS instances, you need to **install Python 3.9** or above (See https://www.python.org/downloads/).

> **Not using the right version of Python could lead Ansible to fail when creating an AWS instance. Please remove the entire virtual env directory (.venv) at the root directory of the project to start from a fresh environment and the right dependencies.**

- For running the tests on supported platforms through CI deployment , **docker should be installed** on your local machine. 

Further, the following variables need to be exported:

- *AWS_SECRET_ACCESS_KEY*: AWS secret access key
- *AWS_ACCESS_KEY_ID*: AWS access key id

Install python deps:

```shell
make -C .ci setup-env
```

It will create a `.runID` under the `.ci` directory. It will contain an unique identifier for your machines, which will be added as a VM tag.

We are able to run the Elastic Stack and all the test suites in AWS instances, so whenever a build failed we will be able to recreate the same machine and access it to inspect its state: logs, files, containers... For that, we are enabling SSH access to those ephemeral machines, which can be created following this guide.

But you must first understand that there are two types of Cloud machines: 
1) the VM running the stack, and 
2) the VMs running the actual tests, where the Elastic Agent will be installed and enrolled into the stack.

#### Running the build scripts outside the Elastic Observability AWS account

In the case you are running the scripts outside the "Elastic Observability" AWS account, please fulfill this requirements before you start creating the instances:

1. Use `us-east-2` (Ohio) as your default AWS region. All the community AMIs that we use are hosted there.
2. Create a "Security Group" named `e2e`. This security group will allow remote access to certain ports in the remote instances we are creating. In this security group please use `0.0.0.0/0` as the Source for the following ports:
   - HTTP 80
   - HTTPS 443
   - SSH 22
   - Elasticsearch 9200
   - Kibana 5601
   - Fleet Server 8220

### Create and configure the stack VM

This specialised VM starts Elasticsearch, Kibana and Fleet Server using Docker Compose, but instead of invoking the compose file directly, it uses the test framework to do it. Why? Because we need to wait for Elasticsearch to be ready and request an API Token to be passed to the Fleet Server container. And [we do this with code](https://github.com/elastic/e2e-testing/blob/4517dfa134844f720139d6bab3955cc8d9c6685c/e2e/_suites/fleet/fleet.go#L631-L748).

The VM is a Debian AMD64 machine, as described [here](https://github.com/elastic/e2e-testing/blob/4517dfa134844f720139d6bab3955cc8d9c6685c/.ci/.e2e-platforms.yaml#L3-L7).

The creation of the stack VM is compounded by two stages: `provision` and `setup`. We separate both stages to be able to provision once, and retry the setup if needed.

To provision and setup the stack node:

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS" # optional, defaults to $(HOME)/.ssh/id_rsa
make -C .ci provision-stack
make -C .ci setup-stack
make -C .ci start-elastic-stack
```

We have created a convenient alias for doing both steps in one command: `create-stack`, which sequentially invokes both of the above commands.

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS" # optional, defaults to $(HOME)/.ssh/id_rsa
make -C .ci create-stack
```

A `.stack-host-ip` file will be created in the `.ci` directory of the project including the IP address of the stack instance. Check it out from that file, or make a note of the IP address displayed in the ansible summary, as you'll probably need it to connect your browser to open Kibana, or to SSH into it for troubleshooting.

> The IP address of the stack in the `.stack-host-ip` file will be used by the automation.

Please remember to [destroy the stack](#destroying-the-stack-and-the-test-nodes) once you finished your testing.

### Create and configure the test node

There are different VM flavours that you can use to run the Elastic Agent and enroll it into the Stack: Debian, CentOS, SLES15, Oracle Linux... using AMD and ARM as architecture. You can find the full reference of the platform support [here](https://github.com/elastic/e2e-testing/blob/4517dfa134844f720139d6bab3955cc8d9c6685c/.ci/.e2e-platforms.yaml#L2-L42).

In these VMs, the test framework will download a binary to install the Elastic Agent (TAR files, DEB/RPM packages...), and will execute the different agent commands to install, enroll, uninstall, etc.


In order to list all the supported platforms, please run this command:

> Docker is needed to run the command, as it internally uses `yq` in the form of a Docker container to read the YML file for the available platforms. To install Docker, please read [this guide](https://docs.docker.com/engine/install/).

```shell
make -C .ci list-platforms 
- stack
- centos8_arm64
- centos8_amd64
- debian_10_arm64
- debian_10_amd64
- debian_11_amd64
- oracle_linux8
- sles15
- fleet_elastic_pkg
- ubuntu_22_04_amd64
- windows2019
```

Once you have the target platform selected, which is obtained from [the platforms descriptor](.e2e-platforms.yaml), you need to pass it to the build script in order to load all the environment variables for the platform. To do so, you only have to add the desired platform as value of the exported `NODE_LABEL` variable:

```shell
# all possible platforms
export NODE_LABEL=centos8_amd64
export NODE_LABEL=centos8_arm64
export NODE_LABEL=debian_10_amd64
export NODE_LABEL=debian_10_arm64
export NODE_LABEL=debian_11_amd64
export NODE_LABEL=debian_11_arm64
export NODE_LABEL=oracle_linux8
export NODE_LABEL=sles15
export NODE_LABEL=ubuntu_22_04_amd64
export NODE_LABEL=windows2019
```

The build will create a `.env-${PLATFORM}` file (i.e. `.env-centos8_arm64`) that will be automatically sourced into your shell before interacting with a test node, so that the environment variables are present for each build command and you do not need to repeat them again and again.

> Important: when running any of the commands below, please check that the `NODE_LABEL` variable is properly set:

```shell
$ env | grep NODE
NODE_LABEL=centos8_arm64
```

Besides that, it's possible to configure the test node for the different test suites that are present in the test framework: `fleet`, `helm` and `kubernetes-autodiscover`. Please configure the test node setting the suite, being `fleet` the default:

```shell
# all possible suites
export SUITE="fleet"
export SUITE="helm"
export SUITE="kubernetes-autodiscover"
```

Next, the creation of the test node is compounded by two stages: `provision` and `setup`. We separate both stages to be able to provision once, and retry the setup if needed.

To provision and setup the test node:

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS" # optional, defaults to $(HOME)/.ssh/id_rsa
make -C .ci provision-node
make -C .ci setup-node
```

We have created a convenient alias for doing both steps in one command: `create-node`, which sequentially invokes both of the above commands.

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS"
export SUITE="fleet"
make -C .ci create-node
```

A `.node-host-ip` file will be created in the `.ci` directory of the project including the IP address of the node instance. Check it out from that file, or make a note of the IP address displayed in the ansible summary, as you'll probably need it to SSH into it for troubleshooting.

> The IP address of the node in that file will be used by the automation.

Please remember to [destroy the node](#destroying-the-stack-and-the-test-nodes) once you have finished your testing.

Finally, start the stack:

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS"
export SUITE="fleet"
make -C .ci start-elastic-stack
```

> You probably need to run this command twice: the Fleet Server could try to start faster than Kibana and die. Running the command again will recreate the container for Fleet Server.

> The `recreate-fleet-server` command has been deprecated, and calls the `start-elastic-stack` command instead.

### Run a test suite

You can select the specific tags that you want to include in the test execution. Please look for the different tags in the existing feature files for the suite you are interested in running. For that, please check out the tags/annotations that are present in those feature files (`*.feature`), which live in the `features` directory under your test suite. In example, for `fleet` test suite, you can find them [here](../e2e/_suites/fleet/features/).

```shell
# example tags
export TAGS="fleet_mode"
export TAGS="system_integration"
export TAGS="apm-server"
export TAGS="kubernetes-autodiscover && elastic-agent"
```

It's important that you consider reading about [the environment variables affecting the build](../e2e/README.md#environment-variables-affecting-the-build), as you could pass them to Make to run the tests with different options, such as a Github commit sha and repo (for testing a PR), the elastic-agent version, for testing a previous version of the agent, to name a few.

Finally, run the tests for non-Windows instances:

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS"
make -C .ci run-tests TAGS="fleet_mode && install"
```

And for Windows instances:

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS"
make -C .ci run-tests-win TAGS="fleet_mode && install"
```

#### Environment variables affecting the build

The following environment variables affect how the tests are run in both the CI and a local machine.

- `BEAT_VERSION`. Set this environment variable to the proper version of the Beats to be used in the current execution. The default value depends on the branch you are targeting your work: See https://github.com/elastic/e2e-testing/blob/70b1d3ddaf39567aeb4c322054b93ad7ce53e825/.ci/Jenkinsfile#L44
- `DEVELOPER_MODE`: Set this environment variable to `true` to activate developer mode, which means not destroying the services provisioned by the test framework. Default: `false`.
- `ELASTIC_AGENT_VERSION`. Set this environment variable to the proper version of the Elastic Agent to be used in the current execution. The default value depends on the branch you are targeting your work: See https://github.com/elastic/e2e-testing/blob/70b1d3ddaf39567aeb4c322054b93ad7ce53e825/.ci/Jenkinsfile#L44
- `ELASTIC_AGENT_DOWNLOAD_URL`. Set this environment variable if you know the bucket URL for an Elastic Agent artifact generated by the CI, i.e. for a pull request. It will take precedence over the `BEAT_VERSION` variable. Default empty: See https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/.ci/Jenkinsfile#L35
- `ELASTIC_APM_ACTIVE`: Set this environment variable to `true` if you want to send instrumentation data to our CI clusters. When the tests are run in our CI, this variable will always be enabled. Default value: `false`.
- `ELASTIC_APM_ENVIRONMENT`: Set this environment variable to `ci` to send APM data to Elastic Cloud. Otherwise, the framework will spin up local APM Server and Kibana instances. For the CI, it will read credentials from Vault. Default value: `local`.
- `FEATURES`: Set this environment variable to an existing feature file, or a glob expression (`fleet_*.feature`), that will be passed to the test runner to filter the execution, selecting those feature files matching that expression. If empty, all feature files in the `features/` directory will be used. It can be used in combination with `TAGS`.
- `GITHUB_CHECK_REPO`: Set this environment variable to the name of the Github repository where the above git SHA commit lives. Default: elastic-agent.
- `GITHUB_CHECK_SHA1`: Set this environment variable to the git commit in the right repository to use the binary snapshots produced by the CI instead of the official releases. The snapshots will be downloaded from a bucket in Google Cloud Storage. This variable is used by the upstream repositories (beats, elastic-agent), when testing the artifacts generated by their packaging jobs. Default: empty.
- `KIBANA_VERSION`. Set this environment variable to the proper version of the Kibana instance to be used in the current execution, which should be used for the Docker tag of the kibana instance. It will refer to an image related to a Kibana PR, under the Observability-CI namespace. Default is empty.
- `LOG_LEVEL`: Set this environment variable to `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR` or `FATAL` to set the log level in the project. Default: `INFO`.
- `SKIP_PULL`: Set this environment variable to prevent the test suite to pull Docker images and/or external dependencies for all components. Default: `false`
- `SKIP_SCENARIOS`: Set this environment variable to `false` if it's needed to include the scenarios annotated as `@skip` in the current test execution, adding that taf to the `TAGS` variable. Default value: `true`.
- `STACK_VERSION`. Set this environment variable to the proper version of the Elasticsearch to be used in the current execution. The default value depends on the branch you are targeting your work.
    - **main (Fleet):** https://github.com/elastic/e2e-testing/blob/0446248bae1ff604219735998841a21a7576bfdd/e2e/_suites/fleet/ingest-manager_test.go#L39
- `TAGS`: Set this environment variable to [a Cucumber tag expression](https://github.com/cucumber/godog#tags), that will be passed to the test runner to filter the execution, selecting those scenarios matching that expresion, across any feature file. It can be used in combination with `FEATURES`.
- `TIMEOUT_FACTOR`: Set this environment variable to an integer number, which represents the factor to be used while waiting for resources within the tests. I.e. waiting for Kibana needs around 30 seconds. Instead of hardcoding 30 seconds, or 3 minutes, in the code, we use a backoff strategy to wait until an amount of time, specific per situation, multiplying it by the timeout factor. With that in mind, we are able to set a higher factor on CI without changing the code, and the developer is able to locally set specific conditions when running the tests on slower machines. Default: `3`.

#### Keeping the elastic-agent running after one scenario

The test framework ensures that the agent is uninstalled and unenrolled after each test scenario, and this is needed to keep each test scenario idempotent. But it's possible to avoid the uninstall + unenroll phase of the elastic-agent if the `DEVELOPER_MODE=true` variable is set.

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS"
make -C .ci run-tests DEVELOPER_MODE=true TAGS="fleet_mode && install"
```

> Please use this capability when running one single test scenario, otherwise you can find unexpected behaviours caused by running multiple agents in the same host.

### Showing current nodes configuration

If you want to check the current IP address, instance types, SSH user of the nodes you are working with, please run the following commands:

```shell
make -C .ci show-stack
make -C .ci show-node
```

### SSH into a remote VM

Once you have created and set up the remote machines with the above instructions, you can SSH into both the stack and the test node machines. In order to do so, you must be allowed to do so first, and for that, please add your Github username in alphabetical order to [this file](../.ci/ansible/github-ssh-keys), keeping a blank line as file ending. For the CI to work, the file including your user must be merged into the project, but for local development you can keep the file in the local state.

> When submitting the pull request with your user to enable the SSH access, please remember to add the right backport labels (ex. `backport-v8.2.0`) so that you will be able to SSH into the CI machines for all supported maintenance branches.

To SSH into the machines, please use the following commands:

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS"
make -C .ci ssh-stack
make -C .ci ssh-node
```

### Destroying the Elastic Stack

Sometimes you need to tear down the Elastic Stack, or recreate the fleet-server, mostly in the case the API Token used for Fleet Server [expired after 1 hour](https://www.elastic.co/guide/en/elasticsearch/reference/current/security-settings.html#token-service-settings).

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS"
make -C .ci destroy-elastic-stack
```

To recreate Fleet Server:
```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS"
make -C .ci start-elastic-stack
```

### Destroying the stack and the test nodes

Do not forget to destroy the stack and nodes once you're done with your tests!

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS"
make -C .ci destroy-stack
make -C .ci destroy-node
```

## Running tests for a pull request on Elastic Agent or Beats

Because we trigger the E2E tests for each Elastic-Agent and Beats PR that is packaged, it's possible to manually trigger it using the user interface of the CI. To achieve it we must navigate to Jenkins and run the tests in the specific branch the original Beats PR is targeting.

>For further information about packaging Beats, please read [Beat's CI docs](https://github.com/elastic/beats/blob/1de27eed058dd074b58c71094c7678b3536251cb/README.md#ci).

To do so:

1. Navigate to Jenkins: https://beats-ci.elastic.co/job/e2e-tests/job/e2e-testing-mbp/
1. Login as a user
2. Select the base branch for the test code: main, 8.3, 7.17, etc.
3. In the left menu, click on `Buid with Parameters`.
4. In the input parameters form, keep the Beat version (for Fleet) as is, to use each branch's default version.
5. In the input parameters form, keep the stack version (for Fleet) as is, to use each branch's default version.
6. In the input parameters form, set the `GITHUB_CHECK_NAME` to `E2E Tests`. This value will appear as the label for the Github check for the E2E tests.
7. In the input parameters form, set the `GITHUB_CHECK_SHA1` to the `SHA1` of the last commit in your pull request. This value will allow us to modify the mergeable status of that commit with the Github check. Besides that, it will set the specific directory in the GCP bucket to look up the CI binaries.
8. In the input parameters form, set the `GITHUB_CHECK_REPO` to `elastic-agent` or `beats`, depending where the aforementioned SHA1 belongs. This is important to look up the binaries in the right GCP bucket.
9. Click the `Build` button at the bottom of the parameters form.

##Troubleshooting

Make sure :
- Python 3.9 or higher is installed.
- Install python3-venv to resolve virtual environment creation errors.
- In case, you are getting errors while creating AWS VMs, ensure you have exported AWS secret and access ids.
- Check 600 permission is provided to id_rsa key files.
- Run list-platforms command and export Node variabe to resolve Node creation errors.
- While creating windows node, we need to run `create-node` command in portions such as `provision-node` and `setup-node`. Also, some times you need to ssh node to create it successfully.
