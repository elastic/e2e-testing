# Get current directory of a Makefile: https://stackoverflow.com/a/23324703
ROOT_DIR:=$(CURDIR)

include ../commons.mk

TEST_TIMEOUT?=5m

GO_IMAGE?='golang'
GO_VERSION='$(shell cat ../.go-version )'
GO_IMAGE_TAG?='stretch'
GOOS?='linux'
GOARCH?='amd64'
LOG_LEVEL?=INFO

.PHONY: build
build:
	docker run --rm \
		-v $(ROOT_DIR):/go/src/github.com/elastic/e2e-testing/cli \
		-w /go/src/github.com/elastic/e2e-testing/cli \
		-e TARGET_OS=$(GOOS) -e TARGET_ARCH=$(GOARCH) -e GO111MODULE=on \
		$(GO_IMAGE):$(GO_VERSION)-$(GO_IMAGE_TAG) \
		scripts/build-cli.sh

.PHONY: install
install:
	go get -v -t ./...

.PHONY: sync-integrations
sync-integrations:
	OP_LOG_LEVEL=${LOG_LEVEL} go run main.go sync integrations --delete

.PHONY: test
test: test-report-setup
	gotestsum --junitfile "$(PWD)/outputs/TEST-unit-cli.xml" --format testname -- -count=1 -timeout=$(TEST_TIMEOUT) ./...
