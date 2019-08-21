FEATURE?=
FLAG?=
FORMAT?=pretty
LOG_LEVEL?=INFO

.PHONY: build-binary
build-binary:
	cd ../cli && go build -o op && mv op ../metricbeat-tests && cd -

.PHONY: elastic-stack
elastic-stack:
	OP_LOG_LEVEL=${LOG_LEVEL} ./op run stack metricbeat

.PHONY: install
install:
	go get -v -t ./...

.PHONY: install-godog
install-godog:
	go get github.com/DATA-DOG/godog/cmd/godog

.PHONY: functional-test
functional-test: install-godog install elastic-stack
	godog --format=${FORMAT} ${FLAG} ${FEATURE}
