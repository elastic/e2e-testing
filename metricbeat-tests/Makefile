FEATURE?=
FLAG?=
FORMAT?=pretty

.PHONY: install
install:
	go get -v -t ./...

.PHONY: install-godog
install-godog:
	go get github.com/DATA-DOG/godog/cmd/godog

.PHONY: functional-test
functional-test: install-godog install
	godog --format=${FORMAT} ${FLAG} ${FEATURE}
