include ../commons.mk
include ./commons-test.mk

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
