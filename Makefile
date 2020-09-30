export PACT_DIR = $(PWD)/pacts
export LOG_DIR = $(PWD)/pact-log
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

pact: export PACT_TEST := true
pact: install-pact
	@echo "--- 🔨Running Consumer Pact tests "
	cd cli && go test -count=1 github.com/elastic/e2e-testing/cli/services -run 'TestPact'
