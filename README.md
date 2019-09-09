[![Build Status](https://apm-ci.elastic.co/buildStatus/icon?job=beats%2Fpoc-metricbeat-tests-poc-mbp%2Fmaster)](https://apm-ci.elastic.co/job/beats/job/poc-metricbeat-tests-poc-mbp/job/master/)

# Environment Provisioner for Observability

## How to run tests
```sh
# Run the UTs for the module cli
$ GO111MODULE=on make -C cli install test
```

## How to run functional tests
```sh
# Run the Functional tests for the module metricbeat-tests
$ GO111MODULE=on make -C metricbeat-tests install functional-test
```
