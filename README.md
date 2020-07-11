[![Build Status](https://beats-ci.elastic.co/buildStatus/icon?job=e2e-tests%2Fe2e-testing-mbp%2Fmaster)](https://beats-ci.elastic.co/job/e2e-tests/job/e2e-testing-mbp/job/master/)

# Environment Provisioner for Observability

## How to run tests
```sh
# Run the UTs for the module cli
$ GO111MODULE=on make -C cli install test
```

## How to run functional tests
```sh
# Run the Functional tests for the module metricbeat
$ GO111MODULE=on make -C e2e install functional-test
```
