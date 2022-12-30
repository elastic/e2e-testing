# Troubleshooting guide

## Diagnosing test failures
The first step in determining the exact failure is to try and reproduce the test run locally, ideally using the DEBUG or TRACE log level to enhance the log output. Once you've done that, look at the output from the test run.

Each test suite's documentation should contain the specifics to run the tests, but it's summarises to executing `go test` or `godog` in the right directory.

### Running the tests on the Cloud machines

> DISCLAIMER: A more specialised version of how to reproduce a cloud deployment can be found [here](../.ci/README.md#running-a-ci-deployment). Although the information here is valid to get the IP addresses of a CI build, we recommend following the more recent guide to troubleshoot a test error.

On CI, we are running the Elastic Stack and all test suites in AWS instances, so whenever a build failed we would need to access those machines and inspect the state of the machine: logs, files, containers... For that, we are enabling SSH access to those ephemeral machines, which will be kept for debugging purpose if and only if the `DESTROY_CLOUD_RESOURCES` environment variable is set at the Jenkinsfile. In the UI of Jenkins, you can enable it using the `DESTROY_CLOUD_RESOURCES` input argument, checking it to true (default is false). After the build finishes, the cloud instances won't be destroyed.

But you must first understand that there are two types of Cloud machines: 
1) the VM running the stack, and 
2) the VMs where the Elastic Agent will be installed and enrolled into the stack.

#### The Stack VM
This specialised VM starts Elasticsearch, Kibana and Fleet Server using Docker Compose, but instead of invoking the compose file directly, it uses the test framework to do it. Why? Because we need to wait for Elasticsearch to be ready and request an API Token to be passed to the Fleet Server container. And [we do this with code](https://github.com/elastic/e2e-testing/blob/4517dfa134844f720139d6bab3955cc8d9c6685c/e2e/_suites/fleet/fleet.go#L631-L748).

The VM is a Debian AMD64 machine, as described [here](https://github.com/elastic/e2e-testing/blob/4517dfa134844f720139d6bab3955cc8d9c6685c/.ci/.e2e-platforms.yaml#L3-L7).

You may need to SSH into this machine to recreate the stack in the case the API Token used for Fleet Server is [expired after 1 hour](https://www.elastic.co/guide/en/elasticsearch/reference/current/security-settings.html#token-service-settings), 

If you want to recreate the stack, please run the following command, which will run the tests but without any valid scenario (see `TAGS` variable):

```shell
# login as root
sudo su -
# stop previous stack
docker-compose -f ~/.op/compose/profiles/fleet/docker-compose.yml down --remove-orphans
# bootstrap the new stack
TAGS="non-existing-tag" TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE DEVELOPER_MODE=true make -C e2e/_suites/fleet functional-test
```

#### The Agent VMs
There different VM flavours that you can use to run the Elastic Agent and enrol it into the Stack: Debian, CentOS, SLES15, Oracle Linux... using AMD and ARM as architecture. You can find the full reference of the platform support [here](https://github.com/elastic/e2e-testing/blob/4517dfa134844f720139d6bab3955cc8d9c6685c/.ci/.e2e-platforms.yaml#L2-L42).

In these VMs, the test framework will download a binary to install the Elastic Agent (TAR files, DEB/RPM packages...), and will execute the different agent commands to install, enroll, uninstall, etc.

#### Getting SSH access to the VMs
To access the machines, you must be allowed to do so first, and for that, please submit a PR adding your Github username in alphabetical order to [this file](../.ci/ansible/github-ssh-keys), keeping a blank line as file ending. The user to access each EC2 used on the tests can be found [here](https://github.com/elastic/e2e-testing/blob/main/.ci/.e2e-platforms.yaml). When submitting the pull request with your user, please remember to add the right backport labels (ex. `backport-v8.2.0`) so that you will be able to SSH into the supported maintenance branches.

To get the IP address of the VMs, please go to the Jenkins BlueOcean UI of the job you manually triggered with `DEVELOPER_MODE=true`, and look up the **Deploy Test Infra** stage. Under its steps, you will see different Ansible executions to provision the VM. Please open any of them and look for any log entry containing an IP address:

```shell
[2022-04-21T05:20:27.133Z] TASK [Gathering Facts] *********************************************************
[2022-04-21T05:20:29.541Z] ok: [3.144.74.102]
```

The IP address of that VM is `3.144.74.102`.

For the agent VMs it's exactly the same, but looking up the **parallel stages** right after the Test Infra. Again, look for any Ansible task containing an IP address, as shown above.

Once you have both IP addresses, one for the stack and one for the agent in the OS/Arch you are interested in, please open two terminals, one for each. Then SSH into the machines using: 
1) the public SSH key you have in your Github account, and 
2) the right user for the machine, as described [here](https://github.com/elastic/e2e-testing/blob/4517dfa134844f720139d6bab3955cc8d9c6685c/.ci/.e2e-platforms.yaml#L2-L42).

An example of how to SSH in the machine, having multiple SSH keys under the `.ssh` directory, and connecting with the `admin` user because it's a Debian machine:

```shell
ssh -i ~/.ssh/id_rsa_elastic.pub -vvvv admin@18.188.242.30
```

#### Running the tests for the Elastic Agent
Once you have SSH'ed into the right agent VM (ex. CentOS 8 ARM), and checked that the stack is in a valid state (the API token of the Fleet Server has not expired, otherwise you must recreate the entire stack), you can run the tests with a simple command, but you must first set the environment so that the agent is able to connect to the remote stack, which lives in another VM. Just source the `.env` file that the test framework created:

```shell
# log in as root
sudo su -
# move to the project directory
cd /home/${USER}/e2e-testing
# load the environment
source .env
# verify variables
env
```

The env should contain those variables needed for enrolling the agent, such as `ELASTICSEARCH_URL` and `FLEET_SERVER_URL`, among others.

Now you can run the tests, specifying the tags you are interested. Please use the tags in the feature files, where the test framework defines one at the top level, for running an entire feature file, and per scenario, so that it's possible to tell the test runner to run one or multiple scenarios. More about Cucumber tags in [here](https://github.com/cucumber/godog#tags).

```shell
# log in as root
sudo su -
# move to the project directory
cd /home/${USER}/e2e-testing
TAGS="system_integration && diskio" TIMEOUT_FACTOR=5 LOG_LEVEL=TRACE PROVIDER=remote make -C e2e/_suites/fleet functional-test
```

- TAGS: it uses a tag from the `system_integration.feature` file, because we want to run just one scenario.
- LOG_LEVEL: keep it as TRACE to see everything.
- TIMEOUT_FACTOR: this number will multiply the default timeouts when waiting for states, such as waiting for an agent to be online. Default is 3 minutes.
- PROVIDER: use `REMOTE` so that it will connect to the remote stack you already provisioned.

More about the environment variables affecting the build [here](https://github.com/elastic/e2e-testing/tree/main/e2e#environment-variables-affecting-the-build), specially if you are debugging a pull-request, where you may need to pass `GITHUB_CHECK_SHA1` and `GITHUB_CHECK_REPO`.

### Tests fail because the product could not be configured or run correctly
This type of failure usually indicates that code for these tests itself needs to be changed. See the sections on how to run the tests locally in the specific test suite.

### One or more scenarios fail
Check if the scenario has an annotation/tag supporting the test runner to filter the execution by that tag. Godog will run those scenarios. For more information about tags: https://github.com/cucumber/godog/#tags

   ```shell
   cd e2e/_suites/YOUR_SUITE
   OP_LOG_LEVEL=TRACE go test -v --godog.tags='@YOUR_ANNOTATION'
   ```

### (For Mac) Docker containers are not healthy

It's important to configure `Docker for Mac` with enough resources (memory and CPU).

To change it, please use Docker UI, go to `Preferences > Resources > Advanced`, and increase the  `memory` and `CPUs`.

### (For Mac) Docker is not able to save files in a temporary directory

It's important to configure `Docker for Mac` to allow it accessing the `/var/folders` directory, as this framework uses Mac's default temporary directory for storing temporary files.

To change it, please use Docker UI, go to `Preferences > Resources > File Sharing`, and add there `/var/folders` to the list of paths that can be mounted into Docker containers. For more information, please read https://docs.docker.com/docker-for-mac/#file-sharing.
<<<<<<< HEAD
=======

### Unable to create AWS VMS
- Ensure you have exported the `AWS_SECRET_ACCESS_KEY`and `AWS_ACCESS_KEY_ID` values.
- Check permissions on id_rsa key files.
- In case, you get errors like `couldn't resolve module/action 'ec2'`, its may be due to ansible version that is no more compatible.
   - Here, you need to degrade ansible version to support already defined profiles. Ansible version 6.6.0 can be used here to resolve above issue.

   ##Steps to be followed:
   - Go to '.venv/bin' folder and activate the python virtual environment.
   - Specify the ansible version in requirement.txt file under `.ci/ansible` folder.
   - Rerun requirements.txt file manually using `pip install -r requirements.txt' command.
>>>>>>> d5541388 (fix: remove Helm Chart tests (#3285))
