CURRENT_UID := $(shell id -u)
CURRENT_GID := $(shell id -g)

# Get current directory of a Makefile: https://stackoverflow.com/a/23324703
ROOT_DIR:=$(CURDIR)

TEST_TIMEOUT?=5m

GO_IMAGE?='golang'
GO_VERSION?='1.13'
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

.PHONY: sync-integrations
sync-integrations:
	docker run --rm \
		-v $(ROOT_DIR):/go/src/github.com/elastic/e2e-testing/cli \
		-v $(HOME)/.op:/root/.op \
		-u "$(CURRENT_UID):$(CURRENT_GID)" \
		-w /go/src/github.com/elastic/e2e-testing/cli \
		-e GO111MODULE=on -e OP_LOG_LEVEL=${LOG_LEVEL} \
		$(GO_IMAGE):$(GO_VERSION)-$(GO_IMAGE_TAG) \
		go run main.go sync integrations --delete 

.PHONY: test
test:
	go test -v -timeout=$(TEST_TIMEOUT) ./...
