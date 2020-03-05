FEATURE?=
FORMAT?=pretty
LOG_INCLUDE_TIMESTAMP?=TRUE
LOG_LEVEL?=INFO
QUERY_MAX_ATTEMPTS?=5
RETRY_TIMEOUT?=3
STACK_VERSION?=
METRICBEAT_VERSION?=
VERSION_VALUE=`cat ../../cli/VERSION.txt`

ifneq ($(FEATURE),)
FEATURE_FLAG=--tags
endif

GO_IMAGE_TAG?='stretch'
GOOS?='linux'
GOARCH?='amd64'

.PHONY: fetch-binary
fetch-binary:
	@$(MAKE) -C ../../cli build
	cp ../../cli/.github/releases/download/$(VERSION_VALUE)/$(GOOS)$(subst amd,,$(GOARCH))-op ./op
	chmod +x ./op

.PHONY: install
install:
	go get -v -t ./...

.PHONY: install-godog
install-godog: export GO111MODULE := on
install-godog:
	go get -v github.com/DATA-DOG/godog/cmd/godog@v0.7.13

.PHONY: functional-test
functional-test: install-godog
	OP_LOG_LEVEL=${LOG_LEVEL} \
	OP_LOG_INCLUDE_TIMESTAMP=${LOG_INCLUDE_TIMESTAMP} \
	OP_QUERY_MAX_ATTEMPTS=${QUERY_MAX_ATTEMPTS} \
	OP_RETRY_TIMEOUT=${RETRY_TIMEOUT} \
	OP_METRICBEAT_VERSION=${METRICBEAT_VERSION} \
	OP_STACK_VERSION=${STACK_VERSION} \
	godog --format=${FORMAT} ${FEATURE_FLAG} ${FEATURE}

.PHONY: run-elastic-stack
run-elastic-stack:
	OP_LOG_LEVEL=${LOG_LEVEL} ./op run stack metricbeat -v ${STACK_VERSION}

.PHONY: shutdown-elastic-stack
shutdown-elastic-stack:
	OP_LOG_LEVEL=${LOG_LEVEL} ./op stop stack metricbeat

.PHONY: sync-integrations
sync-integrations:
	OP_LOG_LEVEL=${LOG_LEVEL} ./op sync integrations --delete
