TEST_TIMEOUT?=5m

.PHONY: install-godog
install-godog:
	go get github.com/DATA-DOG/godog/cmd/godog

.PHONY: install
install:
	go get -v -t ./...

.PHONY: test
test:
	go test -v -timeout=$(TEST_TIMEOUT) ./services

.PHONY: functional-test
functional-test: install-godog
	godog

.PHONY: functional-test-ci
functional-test-ci: install-godog
	godog --format=junit
