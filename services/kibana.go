package services

// NewKibanaService returns a default Kibana service entity
func NewKibanaService(version string, asDaemon bool, elasticsearchService Service) Service {
	inspect, err := elasticsearchService.Inspect()
	if err != nil {
		return nil
	}

	ip := inspect.NetworkSettings.IPAddress

	env := map[string]string{
		"ELASTICSEARCH_HOSTS": "http://" + ip + ":" + elasticsearchService.GetExposedPort(),
	}

	return &DockerService{
		ContainerName: "kibana-" + version,
		Daemon:        asDaemon,
		Env:           env,
		ExposedPort:   5601,
		Image:         "docker.elastic.co/kibana/kibana",
		Name:          "kibana",
		Version:       version,
	}
}
