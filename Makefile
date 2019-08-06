TEST_TIMEOUT?=5m
REPORT?=outputs/godog-report.out
FEATURE?=*
FORMAT?=pretty
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
	godog --format=${FORMAT} -t ${FEATURE} | tee ${REPORT} ; test $${PIPESTATUS[0]} -eq 0
