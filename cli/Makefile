# Get current directory of a Makefile: https://stackoverflow.com/a/23324703
ROOT_DIR:=$(CURDIR)

export PACT_DIR = $(PWD)/../pacts
export LOG_DIR = $(PWD)/../pact-log
export PATH := $(PWD)/pact/bin:$(PATH)
export PATH

TEST_TIMEOUT?=5m

GO_IMAGE?='golang'
GO_VERSION?='$(shell cat ../.go-version )'
GO_IMAGE_TAG?='stretch'
GOOS?='linux'
GOARCH?='amd64'

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

.PHONY: install-pact
install-pact:
	@if [ ! -d pact/bin ]; then\
		echo "--- Installing Pact CLI dependencies";\
		curl -fsSL https://raw.githubusercontent.com/pact-foundation/pact-ruby-standalone/master/install.sh | bash;\
    fi

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

pact: export PACT_TEST := true
pact: install-pact
	@echo "--- 🔨Running Consumer Pact tests "
	go test -count=1 github.com/elastic/e2e-testing/cli/services -run 'TestPact'

.PHONY: test
test:
	go test -v -timeout=$(TEST_TIMEOUT) ./...
