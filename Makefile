TEST_TIMEOUT?=5m
FEATURE?=
FLAG?=
FORMAT?=pretty
.PHONY: install-godog
install-godog:
	cd metricbeat-tests && go get github.com/DATA-DOG/godog/cmd/godog

.PHONY: install
install:
	go get -v -t ./...

.PHONY: test
test:
	go test -v -timeout=$(TEST_TIMEOUT) ./cli/services

.PHONY: functional-test
functional-test: install-godog
	cd metricbeat-tests && godog --format=${FORMAT} ${FLAG} ${FEATURE}
