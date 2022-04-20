# Troubleshooting guide

## Diagnosing test failures
The first step in determining the exact failure is to try and reproduce the test run locally, ideally using the DEBUG or TRACE log level to enhance the log output. Once you've done that, look at the output from the test run.

Each test suite's documentation should contain the specifics to run the tests, but it's summarises to executing `go test` or `godog` in the right directory.

### SSH into the Cloud machines
On CI, we are running the Elastic Stack and all test suites in AWS instances, so whenever a build failed we would need to access those machines and inspect the state of the machine: logs, files, containers... For that, we are enabling SSH access to those ephemeral machines, which will be kept for debugging purpose if and only if the DEVELOPER_MODE environment variable is set at the Jenkinsfile. In the UI of Jenkins, you can enable it using the DEVELOPMENT_MODE input argument, checking it to true (default is false). After the build finishes, the cloud instances won't be destroyed.

To access the machines, you must be allowed to do so first, and for that, please submit a PR adding your Github username in alphabetical order to [this file](../.ci/ansible/github-ssh-keys), keeping a blank line as file ending. The user to access each EC2 used on the tests can be found [here](https://github.com/elastic/e2e-testing/blob/main/.ci/.e2e-platforms.yaml).

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
