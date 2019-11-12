TEST_TIMEOUT?=5m

.PHONY: build
build:
	../.ci/scripts/build-cli.sh

.PHONY: install
install:
	go get -v -t ./...

.PHONY: test
test:
	go test -v -timeout=$(TEST_TIMEOUT) ./...
