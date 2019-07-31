# Environment Provisioner for Observability

A strong requirement by the team is to simplify the already existing tool `apm-integration-tests`, which provisions the environment to run services owned by the team. For that reason, and checking that many teams have the same requirement of starting an environment for local development or for testing, we are going to create a CLI based on the `apm-integration-tests` to provision that local environment.

An example could be to spin up a Docker container representing a MySQL 5.6 instance, or an Apache 2.4 instance to be monitored by Metricbeat. Or to start an Elastic Stack (Elasticsearch + Kibana) in a specific version.

The tool will use Golang's de-facto standard for writing CLIs [`Cobra`](https://github.com/spf13/cobra), which will allow to create a nice experience running commands and subcommands. In that sense, running a service would mean executing:

```sh
# if you are in the Go development world
$ GO111MODULE=on go run main.go run apache -v 2.4
$ GO111MODULE=on go run main.go run mysql -v 5.6

# if you prefer to install the CLI
$ GO111MODULE=on go build -i -o op
$ ./op run apache -v 2.4
$ ./op run mysql -v 5.6
$ ./op run stack -v 7.2.0
```

The tool also provides a way to stop those running services:
```sh
# if you are in the Go development world
$ GO111MODULE=on go run main.go stop apache -v 2.4
$ GO111MODULE=on go run main.go stop mysql -v 5.6

# if you prefer to install the CLI
$ GO111MODULE=on go build -i -o op
$ ./op stop apache -v 2.4
$ ./op stop mysql -v 5.6
$ ./op stop stack -v 7.2.0
```

>By the way, `op` comes from `Observability Provisioner`.