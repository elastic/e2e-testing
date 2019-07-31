TEST_TIMEOUT?=5m

.PHONY: install
install:
	go get -v -t ./...

.PHONY: test
test:
	go test -v -timeout=$(TEST_TIMEOUT) ./services

.PHONY: functional-test
functional-test:
	go get github.com/DATA-DOG/godog/cmd/godog
	godog
