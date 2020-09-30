export PACT_DIR = $(PWD)/pacts
export PACT_LOG_DIR = $(PWD)/pact-log
export PATH := $(PWD)/pact/bin:$(PATH)
export PATH

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

# This target needs the stack (Elasticsearch, Kibana and Package Registry) up-and-running
.PHONY: pact-provider
pact-provider: export PACT_TEST := true
pact-provider: install-pact
	@echo "--- 🔨Running Provider Pact tests "
	cd cli && go test -count=1 -tags=integration github.com/elastic/e2e-testing/cli/services -run "TestPactProvider"
