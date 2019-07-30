package services

// NewElasticsearchService returns a default Elasticsearch service entity
func NewElasticsearchService(version string, asDaemon bool) Service {
	env := map[string]string{
		"bootstrap.memory_lock":  "true",
		"discovery.type":         "single-node",
		"ES_JAVA_OPTS":           "-Xms512m -Xmx512m",
		"xpack.security.enabled": "true",
	}

	return &DockerService{
		ContainerName: "elasticsearch-" + version,
		Daemon:        asDaemon,
		ExposedPort:   9200,
		Env:           env,
		Image:         "docker.elastic.co/elasticsearch/elasticsearch",
		Name:          "elasticsearch",
		NetworkAlias:  "elasticsearch",
		Version:       version,
	}
}
