include ../commons.mk

TEST_TIMEOUT?=5m
LOG_LEVEL?=INFO

.PHONY: install
install:
	go get -v -t ./...

.PHONY: test
test: test-report-setup
	go run gotest.tools/gotestsum --junitfile "$(PWD)/outputs/TEST-unit-cli.xml" --format testname -- -count=1 -timeout=$(TEST_TIMEOUT) ./...
