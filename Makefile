export PACT_DIR = $(PWD)/pacts
export PACT_LOG_DIR = $(PWD)/pact-log
export PATH := $(PWD)/pact/bin:$(PATH)
export PATH

FLEET_KIBANA_CONFIG := $(PWD)/e2e/_suites/ingest-manager/configurations/kibana.config.yml

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

.PHONY: prepare-pact-provider-deps
prepare-pact-provider-deps: install-pact
	@rm -fr ~/.op/compose
	@echo "--- 🚄 Starting Fleet dependencies"
	cd cli && kibanaConfigPath="$(FLEET_KIBANA_CONFIG)" go run main.go run profile ingest-manager

.PHONY: pact-provider
pact-provider: prepare-pact-provider-deps
	@echo "--- ⏸️ Pausing 60 seconds waiting for Kibana to be ready"
	@sleep 60
	@echo "--- 🔨 Running Provider Pact tests"
	cd cli && go test -count=1 -tags=integration github.com/elastic/e2e-testing/cli/services -run "TestPactProvider"

.PHONY: verify-provider
verify-provider: pact-provider
	@echo "--- 🚒 Stopping Fleet dependencies"
	cd cli && go run main.go stop profile ingest-manager
