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
	METRICBEAT_VERSION=${METRICBEAT_VERSION} \
	STACK_VERSION=${STACK_VERSION} \
	godog --format=${FORMAT} ${FEATURE_FLAG} ${FEATURE}

.PHONY: lint
lint:
	@docker run -ti --rm -v $(PWD):/src -w /src gherkin/lint **/*.feature --disable AvoidOutlineForSingleExample,TooClumsy,TooManySteps,TooManyDifferentTags,TooLongStep

.PHONY: notice
notice:
	@echo "Generating NOTICE"
	go mod tidy
	go mod download
	go list -m -json all | go run go.elastic.co/go-licence-detector \
		-includeIndirect \
		-rules ../notice/rules.json \
		-overrides ../notice/overrides.json \
		-noticeTemplate ../notice/NOTICE.txt.tmpl \
		-noticeOut NOTICE.txt \
		-depsOut ""

.PHONY: sync-integrations
sync-integrations:
	OP_LOG_LEVEL=${LOG_LEVEL} ./op sync integrations --delete
