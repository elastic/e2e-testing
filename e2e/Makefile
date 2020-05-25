FEATURE?=
FORMAT?=pretty
LOG_INCLUDE_TIMESTAMP?=TRUE
LOG_LEVEL?=INFO
QUERY_MAX_ATTEMPTS?=5
RETRY_TIMEOUT?=3
STACK_VERSION?=
METRICBEAT_VERSION?=
TEST_TIMEOUT?=20m
VERSION_VALUE=`cat ../cli/VERSION.txt`

GO_IMAGE_TAG?='stretch'
GOOS?='linux'
GOARCH?='amd64'

.PHONY: fetch-binary
fetch-binary:
	@$(MAKE) -C ../cli build
	cp ../cli/.github/releases/download/$(VERSION_VALUE)/$(GOOS)$(subst amd,,$(GOARCH))-op ./op
	chmod +x ./op

.PHONY: install
install:
	go get -v -t ./...

.PHONY: functional-test
functional-test:
	OP_LOG_LEVEL=${LOG_LEVEL} \
	OP_LOG_INCLUDE_TIMESTAMP=${LOG_INCLUDE_TIMESTAMP} \
	OP_QUERY_MAX_ATTEMPTS=${QUERY_MAX_ATTEMPTS} \
	OP_RETRY_TIMEOUT=${RETRY_TIMEOUT} \
	OP_METRICBEAT_VERSION=${METRICBEAT_VERSION} \
	OP_STACK_VERSION=${STACK_VERSION} \
	go test -timeout=$(TEST_TIMEOUT) -v --godog.format=${FORMAT} ${FEATURE}

.PHONY: sync-integrations
sync-integrations:
	OP_LOG_LEVEL=${LOG_LEVEL} ./op sync integrations --delete

.PHONY: test
test:
	go test -v --tags=unit
