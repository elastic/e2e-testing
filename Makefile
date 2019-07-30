TEST_TIMEOUT?=5m

.PHONY: install
install:
	go get -v -t ./...

.PHONY: test
test: install
	go test -v -timeout=$(TEST_TIMEOUT) ./services

.PHONY: functional-test
functional-test: install
	godog
