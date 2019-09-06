package main

import (
	"github.com/DATA-DOG/godog"
	log "github.com/sirupsen/logrus"

	"github.com/elastic/metricbeat-tests-poc/cli/services"
)

var redisService services.Service

func redisIsRunningForMetricbeat(redisVersion string, metricbeatVersion string) error {
	redisService = serviceManager.Build("redis", redisVersion, false)

	redisService.SetNetworkAlias("redis_" + redisVersion + "-metricbeat_" + metricbeatVersion)

	return serviceManager.Run(redisService)
}

func metricbeatIsInstalledAndConfiguredForRedisModule(metricbeatVersion string) error {
	s, err := RunMetricbeatService(metricbeatVersion, redisService)

	metricbeatService = s

	query = ElasticsearchQuery{
		EventModule:    "redis",
		ServiceVersion: redisService.GetVersion(),
	}

	return err
}

func RedisFeatureContext(s *godog.Suite) {
	s.Step(`^Redis "([^"]*)" is running for metricbeat "([^"]*)"$`, redisIsRunningForMetricbeat)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for Redis module$`, metricbeatIsInstalledAndConfiguredForRedisModule)
	s.Step(`^there are no errors in the "([^"]*)" index$`, thereAreNoErrorsInTheIndex)

	s.BeforeScenario(func(interface{}) {
		log.Debug("Before scenario...")
	})
	s.AfterScenario(func(interface{}, error) {
		log.Debug("After scenario...")
	})
}
