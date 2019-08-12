package services

// RunKibanaService runs a Kibana service, connected to an elasticsearch service
func RunKibanaService(version string, asDaemon bool, elasticsearchService Service) Service {
	inspect, err := elasticsearchService.Inspect()
	if err != nil {
		return nil
	}

	ip := inspect.NetworkSettings.IPAddress

	env := map[string]string{
		"ELASTICSEARCH_HOSTS": "http://" + ip + ":" + elasticsearchService.GetExposedPort(),
	}

	serviceManager := NewServiceManager()

	kibana := serviceManager.Build("kibana", version, asDaemon)

	kibana.SetEnv(env)

	return kibana
}
