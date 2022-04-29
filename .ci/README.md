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
    - **platforms**: a list of platforms where the tests will be executed. Valid values are already declared under the `PLATFORMS` object, using the key of the platform as elements in the list. I.e. `["centos8_arm64", "centos8_amd64", "debian_arm64", "debian_amd64", "sles15"]`. Required.

## Running a CI Deployment

### Prereqs

The following variables need to be exported:

- *AWS_SECRET_ACCESS_KEY*: AWS secret access key
- *AWS_ACCESS_KEY_ID*: AWS access key id

Install python deps:

```shell
make -C .ci setup-env
```

It will create a `.runID` under the `.ci` directory. It will contain an unique identifier for your machines, which will be added as a VM tag.

### Create and configure the stack VM

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS"
make -C .ci create-stack
```

A `.stack-host-ip` file will be created in the `.ci` directory of the project including the IP address of the stack instance. Check it out from that file, or make a note of the IP address displayed in the ansible summary, as you'll probably need it to connect your browser to open Kibana, or to SSH into it for troubleshooting.

> The IP address of the stack in the `.stack-host-ip` file will be used by the automation.

Please remember to [destroy the stack](#destroying-the-stack-and-the-test-node) once you finished your testing.

### Create and configure test node

It's possible to configure the test node (OS, architecture), using the values that are already present in [the platforms descriptor](.e2e-platforms.yaml):

```shell
# example for Centos 8 ARM 64
export NODE_IMAGE="ami-01cdc9e8306344fe0"
export NODE_INSTANCE_TYPE="a1.larg"
export NODE_LABEL="centos8_arm64"
export NODE_USER="centos"
```

Besides that, it's possible to configure the test node for the different test suites that are present in the test framework: `fleet`, `helm` and `kubernetes-autodiscover`. Please configure the test node setting the suite, being `fleet` the default:

```shell
# example for Centos 8 ARM 64
export SUITE="fleet"
export SUITE="helm"
export SUITE="kubernetes-autodiscover"
```

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS"
export SUITE="fleet"
make -C .ci create-node
```

A `.node-host-ip` file will be created in the `.ci` directory of the project including the IP address of the node instance. Check it out from that file, or make a note of the IP address displayed in the ansible summary, as you'll probably need it to SSH into it for troubleshooting.

> The IP address of the node in that file will be used by the automation.

Please remember to [destroy the node](#destroying-the-stack-and-the-test-node) once you finished your testing.

### Run a test suite

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS"
export TAGS="fleet_mode_agent" # please check the feature files
make -C .ci run-tests
```

### Destroying the stack and the test node

Do not forget to destroy the stack and nodes you use!

```shell
export SSH_KEY="PATH_TO_YOUR_SSH_KEY_WITH_ACCESS_TO_AWS"
make -C .ci destroy-stack
make -C .ci destroy-node
```
