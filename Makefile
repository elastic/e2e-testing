.PHONY: clean
clean: clean-workspace clean-docker

.PHONY: clean-docker
clean-docker:
	./.ci/scripts/clean-docker.sh || true

.PHONY: clean-workspace
clean-workspace:
	rm -fr ~/.op/compose

.PHONY: e2e-fleet-fleet
e2e-fleet-fleet:
	SUITE="fleet" TAGS="fleet_mode_agent" TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE DEVELOPER_MODE=true $(MAKE) -C e2e functional-test

.PHONY: e2e-fleet-fleet-ci-snapshots
e2e-fleet-fleet-ci-snapshots:
	SUITE="fleet" TAGS="fleet_mode_agent" TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE BEATS_USE_CI_SNAPSHOTS=true DEVELOPER_MODE=true GITHUB_CHECK_SHA1=a1962c8864016010adcde9f35bd8378debb4fbf7 $(MAKE) -C e2e functional-test

.PHONY: e2e-fleet-fleet-pr-ci-snapshots
e2e-fleet-fleet-pr-ci-snapshots:
	SUITE="fleet" TAGS="fleet_mode_agent" TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE BEATS_USE_CI_SNAPSHOTS=true DEVELOPER_MODE=true ELASTIC_AGENT_VERSION=pr-14954 $(MAKE) -C e2e functional-test

.PHONY: e2e-fleet-nightly
e2e-fleet-nightly:
	SUITE="fleet" TAGS="fleet_mode_agent && nightly" TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE DEVELOPER_MODE=true $(MAKE) -C e2e functional-test

.PHONY: e2e-fleet-nightly-ci-snapshots
e2e-fleet-nightly-ci-snapshots:
	SUITE="fleet" TAGS="fleet_mode_agent && nightly" TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE BEATS_USE_CI_SNAPSHOTS=true DEVELOPER_MODE=true GITHUB_CHECK_SHA1=a1962c8864016010adcde9f35bd8378debb4fbf7 $(MAKE) -C e2e functional-test

.PHONY: install
install:
	go get -v -t ./...
