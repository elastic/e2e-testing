TEST_TIMEOUT?=5m
REPORT?=report.xml
FEATURE?=*
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
	cd metricbeat-tests && godog -t ${FEATURE}

.PHONY: functional-test-ci
functional-test-ci: install-godog
	cd metricbeat-tests && godog --format=junit -t ${FEATURE} | tee ${REPORT} ; test $${PIPESTATUS[0]} -eq 0
