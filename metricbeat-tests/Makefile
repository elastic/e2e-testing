FEATURE?=
FLAG?=
FORMAT?=pretty
LOG_LEVEL?=INFO

.PHONY: build-binary
build-binary:
	cd ../cli && go build -o op && mv op ../metricbeat-tests && cd -

.PHONY: install
install:
	go get -v -t ./...

.PHONY: functional-test
functional-test:
	godog --format=${FORMAT} ${FLAG} ${FEATURE}

.PHONY: run-elastic-stack
run-elastic-stack:
	OP_LOG_LEVEL=${LOG_LEVEL} ./op run stack metricbeat

.PHONY: shutdown-elastic-stack
shutdown-elastic-stack:
	OP_LOG_LEVEL=${LOG_LEVEL} ./op stop stack metricbeat
