TEST_TIMEOUT?=5m

# Prepare junit build context
.PHONY: test-report-setup
test-report-setup:
	mkdir -p $(PWD)/outputs
