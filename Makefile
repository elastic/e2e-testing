# Get current directory of a Makefile: https://stackoverflow.com/a/23324703

include ./commons.mk

# Builds cli for all supported platforms
.PHONY: build
build:
	goreleaser --snapshot --skip-publish --rm-dist

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

.PHONY: notice
notice:
	@echo "Generating NOTICE"
	# TODO: Re-enable once new version of go-apm-agent is out
	# go mod tidy
	go mod download
	go list -m -json all | go run go.elastic.co/go-licence-detector \
		-includeIndirect \
		-rules ./notice/rules.json \
		-overrides ./notice/overrides.json \
		-noticeTemplate ./notice/NOTICE.txt.tmpl \
		-noticeOut NOTICE.txt \
		-depsOut ""

.PHONY: unit-test
unit-test: test-report-setup unit-test-dir-internal unit-test-dir-pkg

# See https://pkg.go.dev/gotest.tools/gotestsum/#readme-junit-xml-output
.PHONY: unit-test-suite-%
unit-test-dir-%:
	cd $* && go run gotest.tools/gotestsum --junitfile "$(PWD)/outputs/TEST-unit-$*.xml" --format testname -- -count=1 -timeout=$(TEST_TIMEOUT) ./...
