# Get current directory of a Makefile: https://stackoverflow.com/a/23324703
ROOT_DIR:=$(CURDIR)

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

.PHONY: test
test:
	gotestsum --format testname -- -count=1 -timeout=$(TEST_TIMEOUT) ./...
