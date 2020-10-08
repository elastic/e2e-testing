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
