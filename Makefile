export PACT_DIR = $(PWD)/pacts
export PACT_LOG_DIR = $(PWD)/pact-log
export PATH := $(PWD)/pact/bin:$(PATH)
export PATH

TIMEOUT_FACTOR?=1
WAIT_SECONDS = $(shell expr 30 \* $(TIMEOUT_FACTOR))

FLEET_KIBANA_CONFIG := $(PWD)/e2e/_suites/fleet/configurations/kibana.config.yml

.PHONY: clean
clean: clean-workspace clean-docker

.PHONY: clean-docker
clean-docker:
	./.ci/scripts/clean-docker.sh || true

.PHONY: clean-workspace
clean-workspace:
	rm -fr ~/.op/compose

.PHONY: install
install:
	go get -v -t ./...

.PHONY: install-pact
install-pact:
	@if [ ! -d pact/bin ]; then\
		echo "--- Installing Pact CLI dependencies";\
		curl -fsSL https://raw.githubusercontent.com/pact-foundation/pact-ruby-standalone/master/install.sh | bash;\
    fi

.PHONY: pact-consumer
pact-consumer: export PACT_TEST := true
pact-consumer: install-pact
	@echo "--- 🔨Running Consumer Pact tests "
	cd cli && go test -count=1 github.com/elastic/e2e-testing/cli/services -run 'TestPactConsumer'

.PHONY: destroy-pact-provider-deps
destroy-pact-provider-deps:
	@echo "--- 🚒 Stopping Fleet dependencies"
	cd cli && go run main.go stop profile fleet

.PHONY: prepare-pact-provider-deps
prepare-pact-provider-deps: install-pact
	@rm -fr ~/.op/compose
	@echo "--- 🚄 Starting Fleet dependencies"
	cd cli && kibanaConfigPath="$(FLEET_KIBANA_CONFIG)" go run main.go run profile fleet

.PHONY: pact-provider
pact-provider: prepare-pact-provider-deps
	@echo "--- ⏸️ Pausing $(WAIT_SECONDS) seconds waiting for Kibana to be ready"
	@sleep $(WAIT_SECONDS)
	@echo "--- 🔨 Running Provider Pact tests"
	cd cli && go test -count=1 -tags=integration github.com/elastic/e2e-testing/cli/services -run "TestPactProvider"

.PHONY: verify-provider
verify-provider: pact-provider destroy-pact-provider-deps
