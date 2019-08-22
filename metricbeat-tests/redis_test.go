package main

import (
	"github.com/DATA-DOG/godog"
	"github.com/elastic/metricbeat-tests-poc/cli/services"
)

var redisService services.Service

func redisIsRunning(redisVersion string) error {
	redisService = serviceManager.Build("redis", redisVersion, false)

	return serviceManager.Run(redisService)
}

func metricbeatIsInstalledAndConfiguredForRedisModule(metricbeatVersion string) error {
	s, err := RunMetricbeatService(metricbeatVersion, redisService)

	metricbeatService = s

	query = ElasticsearchQuery{
		EventModule: "redis",
	}

	return err
}

func RedisFeatureContext(s *godog.Suite) {
	s.Step(`^Redis "([^"]*)" is running$`, redisIsRunning)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for Redis module$`, metricbeatIsInstalledAndConfiguredForRedisModule)
	s.Step(`^there are no errors in the "([^"]*)" index$`, thereAreNoErrorsInTheIndex)
}
