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
