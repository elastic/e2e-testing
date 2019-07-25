TEST_TIMEOUT?=5m

.PHONY: install
install:
	go get -v -t ./...

.PHONY: test
test:
	godog
