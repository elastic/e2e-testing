# Environment Provisioner for Observability

A strong requirement by the team is to simplify the already existing tool `apm-integration-tests`, which provisions the environment to run services owned by the team. For that reason, and checking that many teams have the same requirement of starting an environment for local development or for testing, we are going to create a CLI based on the `apm-integration-tests` to provision that local environment.

An example could be to spin up a Docker container representing a MySQL 5.6 instance, or an Apache 2.4 instance to be monitored by Metricbeat. Or to start an Elastic Stack (Elasticsearch + Kibana) in a specific version.

The tool will use Golang's de-facto standard for writing CLIs [`Cobra`](https://github.com/spf13/cobra), which will allow to create a nice experience running commands and subcommands. In that sense, running a service would mean executing:

```sh
# if you are in the Go development world
$ GO111MODULE=on go run main.go run service apache -v 2.4
$ GO111MODULE=on go run main.go run service mysql -v 5.6

# if you prefer to install the CLI
$ GO111MODULE=on go build -i -o op
$ ./op run service apache -v 2.4
$ ./op run service mysql -v 5.6
$ ./op run stack observability
```

The tool also provides a way to stop those running services:
```sh
# if you are in the Go development world
$ GO111MODULE=on go run main.go stop service apache -v 2.4
$ GO111MODULE=on go run main.go stop service mysql -v 5.6

# if you prefer to install the CLI
$ GO111MODULE=on go build -i -o op
$ ./op stop service apache -v 2.4
$ ./op stop service mysql -v 5.6
$ ./op stop stack observability
```

>By the way, `op` comes from `Observability Provisioner`.

## Configuring the CLI
The CLI uses a set of YAML files where the configuration is stored, defining a precedence in those files so that it's possible to override or append new configurations to the CLI.

The **default configuration file** is located under `$HOME/.op`, which is the workspace for the tool. If this directory does not exist when the CLI is run, it will create it under the hood, populating the configuration file (`config.yml`) with the default values.

>Those default values are defined at [config.go](./config/config.go).

A **second layer for the configuration file** is defined by CLI's execution path, so if a `config.yml` exists at the location where the tool is executed, then the CLI will merge this file into the default one, with higher precedence over defaults.

**Last layer for the configuration file** is defined by the `OP_CONFIG_PATH` environment variable, so if a `config.yml` exists at the location defined by that environment variable, then the CLI will merge this file into the previous ones, with higher precedence over them.

This way, the configuration precedence is defined by:

`$HOME/.op/config.yml < $(pwd)/config.yml < $OP_CONFIG_PATH/config.yml`

A clear benefit of this layered configuration is to be able to define custom services/stacks at the higher layers of the configuration: the user could define a service or a stack just for that execution, which could be shared accross teams simply copying the configuration file in the proper path. If that configuration is valuable enough, it could be contributed to the tool.

## Why this tool is not building software dependencies

One common issue we have seen across Observability projects is related to the constant need for a project consumer of building Docker images for most of its dependencies (metricbeat building integrations, apm-integration-tests building opbeans, etc.)

We believe that the responsibility of creating those Docker images must fall under the software producer instead of the consumer. And we believe this because a consumer software doesn't have to know how to build its dependencies, it just consumes them in the form of artefacts. Those artefacts could be Docker images, binaries (Go binaries, JAR files, etc) or packages (NPM), and they might be created on their own repositories.

A dependency is identified by its coordinates:
    - Docker: qualifies namespace + tag
    - Java, NPM: qualified namespace + version
    - Go (before modules): Github repository
    - Go (after modules): Github repository + version

If we are able to produce qualified artefacts from each repository that produces a dependency, then we will be able to consume them in an easy manner.

### What are the benefits of consuming instead of building dependencies?

The three biggest benefits are: Developer experience, Build time and Cloud cost.

> As an example, we estimate that metricbeat uses 20% of its build time building Docker images. Let's use the benefits below to understand the impact of applying this strategy to a real project.

#### Developer Experience
Projects that have to build a dependency are harder to maintain, because the way a dependency is built could change. It's true that Docker simplifies the process a lot, but the Dockerfile used to build it could change in the same manner. So the consumer software is forced to update its code to update the build process for each dependency.

If we are able to remove this build part on a consumer, then the code base would be simpler, increasing how it's used by developers.

#### Build time
Projects that have to build a dependency uses more CI time, as they have to build the dependencies each time the CI triggers a job, be it a PR or branch. It's true that Jenkins (an other CI providers) allows to skip stages depending on conditions, but in the end we always need to build the dependencies at some point (probably at the merge to master).

If we are able to remove this build part on a consumer, then the CI build will take less time to finish, as we will consume already published artefacts instead of building them. The pull time will be taken into consideration too, but it would be way faster than the build time.

#### Cloud costs
For the same reason as the previous one, less build time means less life time for a ephemeral build agent on CI, so the cloud costs of running them will decrease in the same percentage as the build time.