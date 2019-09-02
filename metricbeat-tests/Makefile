FEATURE?=
FLAG?=
FORMAT?=pretty
LOG_INCLUDE_TIMESTAMP?=TRUE
LOG_LEVEL?=INFO
METRICBEAT_FETCH_TIMEOUT?=20
QUERY_MAX_ATTEMPTS?=5
RETRY_TIMEOUT?=3

.PHONY: build-binary
build-binary:
	cd ../cli && go build -o op && mv op ../metricbeat-tests && cd -

.PHONY: install
install:
	go get -v -t ./...

.PHONY: functional-test
functional-test:
	OP_LOG_LEVEL=${LOG_LEVEL} \
	OP_LOG_INCLUDE_TIMESTAMP=${LOG_INCLUDE_TIMESTAMP} \
	OP_METRICBEAT_FETCH_TIMEOUT=${METRICBEAT_FETCH_TIMEOUT} \
	OP_QUERY_MAX_ATTEMPTS=${QUERY_MAX_ATTEMPTS} \
	OP_RETRY_TIMEOUT=${RETRY_TIMEOUT} \
	godog --format=${FORMAT} ${FLAG} ${FEATURE}

.PHONY: run-elastic-stack
run-elastic-stack:
	OP_LOG_LEVEL=${LOG_LEVEL} ./op run stack metricbeat

.PHONY: shutdown-elastic-stack
shutdown-elastic-stack:
	OP_LOG_LEVEL=${LOG_LEVEL} ./op stop stack metricbeat
