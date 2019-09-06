package main

import (
	"github.com/DATA-DOG/godog"
	log "github.com/sirupsen/logrus"
)

var redisTestSuite MetricbeatTestSuite

func redisIsRunningForMetricbeat(redisVersion string, metricbeatVersion string) error {
	redisService := serviceManager.Build("redis", redisVersion, true)

	redisService.SetNetworkAlias("redis_" + redisVersion + "-metricbeat_" + metricbeatVersion)

	err := serviceManager.Run(redisService)
	if err == nil {
		redisTestSuite.Service = redisService
	}

	return err
}

func metricbeatIsInstalledAndConfiguredForRedisModule(metricbeatVersion string) error {
	redisService := redisTestSuite.Service

	s, err := RunMetricbeatService(metricbeatVersion, redisService)
	if err == nil {
		redisTestSuite.Metricbeat = s
	}

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
		redisTestSuite = MetricbeatTestSuite{}
	})
	s.AfterScenario(func(interface{}, error) {
		log.Debug("After scenario...")
		redisTestSuite.CleanUp()
	})
}
