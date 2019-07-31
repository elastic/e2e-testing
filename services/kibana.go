package services

// NewKibanaService returns a default Kibana service entity
func NewKibanaService(version string, asDaemon bool) Service {
	return &DockerService{
		ContainerName: "kibana-" + version,
		Daemon:        asDaemon,
		ExposedPort:   5601,
		Image:         "docker.elastic.co/kibana/kibana",
		Name:          "kibana",
		NetworkAlias:  "kibana",
		Version:       version,
	}
}

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

	kibana := NewKibanaService(version, asDaemon)

	kibana.SetEnv(env)

	return kibana
}
