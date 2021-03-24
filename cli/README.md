# Environment Provisioner for Observability

A strong requirement by the team is to simplify the already existing tool `apm-integration-tests`, which provisions the environment to run services owned by the team. For that reason, and checking that many teams have the same requirement of starting an environment for local development or for testing, we are going to create a CLI based on the `apm-integration-tests` to provision that local environment.

An example could be to spin up a Docker container representing a MySQL 5.6 instance, or an Apache 2.4 instance to be monitored by Metricbeat. Or to start an Elastic Stack (Elasticsearch + Kibana) in a specific version.

The tool will use Golang's de-facto standard for writing CLIs [`Cobra`](https://github.com/spf13/cobra), which will allow to create a nice experience running commands and subcommands. In that sense, running a service would mean executing:

```sh
# if you are in the Go development world
$ GO111MODULE=on go run main.go run service apache -v 2.4.20
$ GO111MODULE=on go run main.go run service mysql -v 5.7.12

# if you prefer to install the CLI
$ GO111MODULE=on go build -i -o op
$ ./op run service apache -v 2.4.20
$ ./op run service mysql -v 5.7.12
$ ./op run profile metricbeat
```

The tool also provides a way to stop those running services:
```sh
# if you are in the Go development world
$ GO111MODULE=on go run main.go stop service apache -v 2.4.20
$ GO111MODULE=on go run main.go stop service mysql -v 5.7.12

# if you prefer to install the CLI
$ GO111MODULE=on go build -i -o op
$ ./op stop service apache -v 2.4.20
$ ./op stop service mysql -v 5.7.12
$ ./op stop profile metricbeat
```

>By the way, `op` comes from `Observability Provisioner`.

## Configuring the CLI
The **default configuration file** is located under `$HOME/.op`, which is the workspace for the tool. If this directory does not exist when the CLI is run, it will create it under the hood, populating the default services already bundled in the tool.

>Those default services are defined at [config.go](./config/config.go).

### Updating services from Beats
The CLI includes a command to fetch Beats integrations from its GitHub repository, making it possible to add them to the list of available services. To run this command:

```
$ ./op sync integrations -h
Sync services from Beats, checking out current version of the services from GitHub

Usage:
  op sync integrations [flags]

Flags:
  -h, --help            help for integrations
  -d, --delete          Will delete the existing Beats repository before cloning it again (default false)
  -r, --remote string   Sets the remote for Beats, using 'user:branch' as format (i.e. elastic:master) (default "elastic:master")
```

It's possible to update the services from a different remote, using the `--remote` flag, as described above.

## Logging
The CLI uses [`Logrus`](https://github.com/sirupsen/logrus) as default Logger, so it's possible to configure the logger using [Logging levels](https://github.com/sirupsen/logrus#level-logging) to enrich the output of the tool.

To set the log level, please set the environment variable `OP_LOG_LEVEL` to one of the following values, being `INFO` the default one:

```
$ export OP_LOG_LEVEL=TRACE
$ export OP_LOG_LEVEL=DEBUG
$ export OP_LOG_LEVEL=INFO
$ export OP_LOG_LEVEL=WARNING
$ export OP_LOG_LEVEL=ERROR
$ export OP_LOG_LEVEL=FATAL
$ export OP_LOG_LEVEL=PANIC
```

It's also possible to add current timestamp to each log line. To do so, please set the environment variable `OP_LOG_INCLUDE_TIMESTAMP`, with "true" (case-insensitive).

```
$ export OP_LOG_INCLUDE_TIMESTAMP=TRUE
$ export OP_LOG_INCLUDE_TIMESTAMP=true
```

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
