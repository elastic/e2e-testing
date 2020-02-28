TEST_TIMEOUT?=5m

GO_IMAGE?='golang'
GO_VERSION?='1.13'
GO_IMAGE_TAG?='stretch'
GOOS?='linux'
GOARCH?='amd64'

.PHONY: build
build:
	docker run --rm \
		-v $(PWD):/go/src/github.com/elastic/metricbeat-tests-poc/cli \
		-w /go/src/github.com/elastic/metricbeat-tests-poc/cli \
		-e TARGET_OS=$(GOOS) -e TARGET_ARCH=$(GOARCH) -e GO111MODULE=on \
		$(GO_IMAGE):$(GO_VERSION)-$(GO_IMAGE_TAG) \
		scripts/build-cli.sh

.PHONY: install
install:
	go get -v -t ./...

.PHONY: test
test:
	go test -v -timeout=$(TEST_TIMEOUT) ./...
