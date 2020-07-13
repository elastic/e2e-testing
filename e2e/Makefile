SUITE?=metricbeat
FEATURE?=
FORMAT?=pretty
LOG_INCLUDE_TIMESTAMP?=TRUE
LOG_LEVEL?=INFO
RETRY_TIMEOUT?=3
STACK_VERSION?=
METRICBEAT_VERSION?=
VERSION_VALUE=`cat ../cli/VERSION.txt`

ifneq ($(FEATURE),)
FEATURE_FLAG=--tags
endif

GO_IMAGE_TAG?='stretch'

.PHONY: install
install:
	go get -v -t ./...

.PHONY: install-godog
install-godog: export GO111MODULE := on
install-godog:
	go get -v github.com/cucumber/godog/cmd/godog@v0.9.0

.PHONY: functional-test
functional-test: install-godog
	cd _suites/${SUITE} && \
	OP_LOG_LEVEL=${LOG_LEVEL} \
	OP_LOG_INCLUDE_TIMESTAMP=${LOG_INCLUDE_TIMESTAMP} \
	OP_RETRY_TIMEOUT=${RETRY_TIMEOUT} \
	OP_METRICBEAT_VERSION=${METRICBEAT_VERSION} \
	OP_STACK_VERSION=${STACK_VERSION} \
	godog --format=${FORMAT} ${FEATURE_FLAG} ${FEATURE}
