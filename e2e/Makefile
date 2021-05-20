include ../commons.mk
include ./commons-test.mk

PICKLES_VERSION?="2.20.1"
VERSION_VALUE=`cat ../cli/VERSION.txt`

.PHONT: build-docs
build-docs:
	rm -fr docs
	@docker run --rm --user $$(id -u):$$(id -g) -v $(PWD):/suites docker.elastic.co/observability-ci/picklesdoc:$(PICKLES_VERSION) -f /suites -o /suites/docs --sn "E2E Testing" --sv $(VERSION_VALUE)
	# because pickledocs is a Windows tool, there is a wrong slash.
	mv docs/.\\/index.html docs/index.html
	rm -fr docs/.\\

## Test examples

.PHONY: fleet-fleet
fleet-fleet:
	TAGS="fleet_mode_agent" TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE DEVELOPER_MODE=true $(MAKE) -C _suites/fleet functional-test

.PHONY: fleet-fleet-ci-snapshots
fleet-fleet-ci-snapshots:
	TAGS="fleet_mode_agent" TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE BEATS_USE_CI_SNAPSHOTS=true DEVELOPER_MODE=true GITHUB_CHECK_SHA1=a1962c8864016010adcde9f35bd8378debb4fbf7 $(MAKE) -C _suites/fleet functional-test

.PHONY: fleet-nightly
fleet-nightly:
	TAGS="fleet_mode_agent && nightly" TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE DEVELOPER_MODE=true $(MAKE) -C _suites/fleet functional-test

.PHONY: fleet-nightly-ci-snapshots
fleet-nightly-ci-snapshots:
	TAGS="fleet_mode_agent && nightly" TIMEOUT_FACTOR=3 LOG_LEVEL=TRACE BEATS_USE_CI_SNAPSHOTS=true DEVELOPER_MODE=true GITHUB_CHECK_SHA1=a1962c8864016010adcde9f35bd8378debb4fbf7 $(MAKE) -C _suites/fleet functional-test
