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
