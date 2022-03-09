# CI considerations

## Jenkins Stages and platform support
There are a set of YAML files, named after `.e2e-tests*.yaml`, that drive the execution of the different test scenarios and suites in the CI. These files are read by Jenkins at build time creating as many Jenkins parallel branch as items declared in the files. Ideally, each parallel branch should run a set of tests in a specific platform, i.e. the _"apm-server scenarios on Centos 8"_.

### Available platforms file structure
To support multiple test suite descriptors, we are defining the available platforms in one single file: `.e2e-platforms.yaml`. The structure of the file is the following:

- **PLATFORMS**: this entry will hold a YAML object will all the available platforms where the tests could be run. Each available platform will be defined as a key in the YAML object, where each object will have a key identifying the platform with a very descriptiva name (i.e. `stack` or `debian_arm64`) and the following attributes:
  - **description**: Description of the purpose and/or characteristics of the platform machine. Required.
  - **image**: the AWS AMI identifier, i.e. `ami-0d90bed76900e679a`. Required.
  - **instance_type**: the AWS instance type, representing the size of the machine, i.e. `c5.4xlarge`. Required.
  - **username**: the default user name when connecting to the machine using SSH, used by Ansible to execute commands on the remote machine. I.e. `centos`, `admin` or `ec2-user`. Required.

In order to configure each platform, there is an `Ansible` script that installs the runtime dependencies for the machine, such as specific OS packages or libraries. Any available platform must declare its own block in the [`./ansible/tasks/install_deps.yml`](./ansible/tasks/install_deps.yml) file in order to be considered as supported.

### Test suites and scenarios file structure
It's possible that a consumer of the e2e tests would need to define a specific layout for the test execution, adding or removing suites and/or scenarios. That's the case for Beats or the Elastic Agent, which triggers the E2E tests with a different layout than for the own development of the test framework: while in Beats or the Elastic Agent we are more interested in running the test for Fleet only, when developing the project we want to verify all the test suites at a time. The structure of these files is the following:

- **SUITES**: this entry will hold a YAML object containing a list of suite. Each suite in the list will be represented by a YAML object with the following attributes:
  - **suite**: the name of the suite. WIll be used to look up the root directory of the test suite, located under the `e2e/_suites` directory. Therefore, only `fleet`, `helm` and `kubernetes-autodiscover` are valid values. Required.
  - **provider**: declares the provider type for the test suite. Valid values are `docker`, `elastic-package` and `remote`. If not present, it will use `remote` as fallback. Optional.
  - **scenarios**: a list of YAML objects representing the test scenarios, where the tests are executed. A test scenario will basically declare how to run a set of test, using the following attributes:
    - **name**: name of the test scenario. It will be used by Jenkins to name the parallel stage representing this scenario. Required.
    - **provider**: declares the provider type for the test scenario. Valid values are `docker`, `elastic-package` and `remote`. If not present, it will use its parent test suite's provider. Optional.
    - **tags**: a Gherkin expression to filter scenarios by tag. It will drive the real execution of the tests, selecting which feature files and/or Cucumber tags will be added to the current test execution. An example could be `linux_integration` or `running_on_beats`. For reference, see https://github.com/cucumber/godog#tags. Required.
    - **platforms**: a list of platforms where the tests will be executed. Valid values are already declared under the `PLATFORMS` object, using the key of the platform as elements in the list. I.e. `["centos8_arm64", "centos8_amd64", "debian_arm64", "debian_amd64", "sles15"]`. Required.

## Running a CI Deployment

### Prereqs

The following variables need to be exported:

- *RUN_ID*: This is a unique identifying ID for the current run. It can be an arbitrary name or something like this:

```
export RUN_ID=$(uuidgen|cut -d'-' -f1)
```

- *AWS_SECRET_ACCESS_KEY*: AWS secret access key
- *AWS_ACCESS_KEY_ID*: AWS access key id

Install python deps:

```
> python3 -mvenv venv
> venv/bin/pip3 install ansible requests boto3 boto
> venv/bin/ansible-galaxy install -r .ci/ansible/requirements.yml
```

### Deploy stack

```
> venv/bin/ansible-playbook .ci/ansible/playbook.yml \
    --private-key="$HOME/.ssh/id_rsa" \
    --extra-vars "nodeLabel=stack nodeImage=ami-0d90bed76900e679a nodeInstanceType=c5.4xlarge" \
    --extra-vars "runId=$RUN_ID workspace=$HOME/Projects/e2e-testing/ sshPublicKey=$HOME/.ssh/id_rsa.pub" \
    --ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
    -t provision-stack
```

Make note of the IP address displayed in the ansible summary.

### Setup stack

```
> venv/bin/ansible-playbook .ci/ansible/playbook.yml \
    --private-key="$HOME/.ssh/id_rsa" \
    --extra-vars "nodeLabel=stack nodeImage=ami-0d90bed76900e679a nodeInstanceType=c5.4xlarge" \
    --extra-vars "runId=$RUN_ID workspace=$HOME/Projects/e2e-testing/ sshPublicKey=$HOME/.ssh/id_rsa.pub" \
    --ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
    -t setup-stack \
    -i <ip address above>,
```

**Note**: The comma at the end of the ip address is required.

### Deploy test node

```
> venv/bin/ansible-playbook .ci/ansible/playbook.yml \
    --private-key="$HOME/.ssh/id_rsa" \
    --extra-vars "stackRunner=<ip address from above> nodeLabel=fleet_amd64 nodeImage=ami-0d90bed76900e679a nodeInstanceType=c5.4xlarge" \
    --extra-vars "runId=$RUN_ID workspace=$HOME/Projects/e2e-testing/ sshPublicKey=$HOME/.ssh/id_rsa.pub" \
    --ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
    -t setup-stack
```

Make note of the ip address displayed in the ansible summary.

### Setup test node

```
> venv/bin/ansible-playbook .ci/ansible/playbook.yml \
    --private-key="$HOME/.ssh/id_rsa" \
    --extra-vars "stackRunner=<ip address from above> nodeLabel=fleet_amd64 nodeImage=ami-0d90bed76900e679a nodeInstanceType=c5.4xlarge" \
    --extra-vars "runId=$RUN_ID workspace=$HOME/Projects/e2e-testing/ sshPublicKey=$HOME/.ssh/id_rsa.pub" \
    --ssh-common-args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
    -t setup-node \
    -i <ip address of node from above>,
```

**Note**: The comma at the end of the ip address is required.

### Run a test suite

```
> ssh -i $HOME/.ssh/id_rsa admin@<node ip address>
node> sudo bash e2e-testing/.ci/scripts/functional-test.sh "fleet_mode_agent"
```
