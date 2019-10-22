FEATURE?=
FORMAT?=pretty
LOG_INCLUDE_TIMESTAMP?=TRUE
LOG_LEVEL?=INFO
METRICBEAT_FETCH_TIMEOUT?=20
QUERY_MAX_ATTEMPTS?=5
RETRY_TIMEOUT?=3
STACK_VERSION?=
METRICBEAT_VERSION?=

ifneq ($(FEATURE),)
FEATURE_FLAG=--tags
endif

.PHONY: build-binary
build-binary:
	cd ../cli && go build -o op && mv op ../metricbeat-tests && cd -

.PHONY: install
install:
	go get -v -t ./...

.PHONY: install-godog
install-godog: export GO111MODULE := on
install-godog:
	mkdir -p temp && cd temp && go mod init && go get -d -v github.com/DATA-DOG/godog@v0.7.13 && cd .. && rm -fr temp

.PHONY: functional-test
functional-test: install-godog
	OP_LOG_LEVEL=${LOG_LEVEL} \
	OP_LOG_INCLUDE_TIMESTAMP=${LOG_INCLUDE_TIMESTAMP} \
	OP_METRICBEAT_FETCH_TIMEOUT=${METRICBEAT_FETCH_TIMEOUT} \
	OP_QUERY_MAX_ATTEMPTS=${QUERY_MAX_ATTEMPTS} \
	OP_RETRY_TIMEOUT=${RETRY_TIMEOUT} \
	OP_METRICBEAT_VERSION=${METRICBEAT_VERSION} \
	godog --format=${FORMAT} ${FEATURE_FLAG} ${FEATURE}

.PHONY: run-elastic-stack
run-elastic-stack:
	OP_LOG_LEVEL=${LOG_LEVEL} ./op run stack metricbeat -v ${STACK_VERSION}

.PHONY: shutdown-elastic-stack
shutdown-elastic-stack:
	OP_LOG_LEVEL=${LOG_LEVEL} ./op stop stack metricbeat
