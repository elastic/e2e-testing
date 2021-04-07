# Get current directory of a Makefile: https://stackoverflow.com/a/23324703
ROOT_DIR:=$(CURDIR)

include ../commons.mk

TEST_TIMEOUT?=5m
LOG_LEVEL?=INFO

.PHONY: install
install:
	go get -v -t ./...

.PHONY: sync-integrations
sync-integrations:
	OP_LOG_LEVEL=${LOG_LEVEL} go run main.go sync integrations --delete

.PHONY: test
test: test-report-setup
	gotestsum --junitfile "$(PWD)/outputs/TEST-unit-cli.xml" --format testname -- -count=1 -timeout=$(TEST_TIMEOUT) ./...
