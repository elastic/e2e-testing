package main

import "github.com/cucumber/godog"

func IngestManagerFeatureContext(s *godog.Suite) {
	s.Step(`^the "([^"]*)" Kibana setup has been executed$`, theKibanaSetupHasBeenExecuted)
	s.Step(`^an agent is deployed to Fleet$`, anAgentIsDeployedToFleet)
	s.Step(`^the agent is listed in Fleet as online$`, theAgentIsListedInFleetAsOnline)
	s.Step(`^system package dashboards are listed in Fleet$`, systemPackageDashboardsAreListedInFleet)
	s.Step(`^there is data in the index$`, thereIsDataInTheIndex)
	s.Step(`^"([^"]*)" is "([^"]*)" on the host$`, isOnTheHost)
	s.Step(`^the agent is "([^"]*)" on the host$`, theAgentIsOnTheHost)
	s.Step(`^the agent is un-enrolled$`, theAgentIsUnenrolled)
	s.Step(`^the agent is not listed as online in Fleet$`, theAgentIsNotListedAsOnlineInFleet)
	s.Step(`^there is no data in the index$`, thereIsNoDataInTheIndex)
	s.Step(`^an agent is enrolled$`, anAgentIsEnrolled)
	s.Step(`^the agent is re-enrolled on the host$`, theAgentIsReenrolledOnTheHost)
	s.Step(`^"([^"]*)" is listed in Fleet as online$`, isListedInFleetAsOnline)
	s.Step(`^the enrollment token is revoked$`, theEnrollmentTokenIsRevoked)
	s.Step(`^an attempt to enroll a new agent fails$`, anAttemptToEnrollANewAgentFails)
}

func theKibanaSetupHasBeenExecuted(arg1 string) error {
	return godog.ErrPending
}

func anAgentIsDeployedToFleet() error {
	return godog.ErrPending
}

func theAgentIsListedInFleetAsOnline() error {
	return godog.ErrPending
}

func systemPackageDashboardsAreListedInFleet() error {
	return godog.ErrPending
}

func thereIsDataInTheIndex() error {
	return godog.ErrPending
}

func isOnTheHost(arg1, arg2 string) error {
	return godog.ErrPending
}

func theAgentIsOnTheHost(arg1 string) error {
	return godog.ErrPending
}

func theAgentIsUnenrolled() error {
	return godog.ErrPending
}

func theAgentIsNotListedAsOnlineInFleet() error {
	return godog.ErrPending
}

func thereIsNoDataInTheIndex() error {
	return godog.ErrPending
}

func anAgentIsEnrolled() error {
	return godog.ErrPending
}

func theAgentIsReenrolledOnTheHost() error {
	return godog.ErrPending
}

func isListedInFleetAsOnline(arg1 string) error {
	return godog.ErrPending
}

func theEnrollmentTokenIsRevoked() error {
	return godog.ErrPending
}

func anAttemptToEnrollANewAgentFails() error {
	return godog.ErrPending
}
