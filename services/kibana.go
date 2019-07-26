package services

import "github.com/docker/go-connections/nat"

// NewKibanaService returns a default Kibana service entity
func NewKibanaService(version string, asDaemon bool, elasticsearchService Service) Service {
	inspect, err := elasticsearchService.Inspect()
	if err != nil {
		return nil
	}

	p := elasticsearchService.GetExposedPort() + "/tcp"
	ip := inspect.NetworkSettings.IPAddress
	portsMap := inspect.NetworkSettings.Ports
	pm := portsMap[nat.Port(p)][0]

	env := map[string]string{
		"ELASTICSEARCH_HOSTS": "http://" + ip + ":" + pm.HostPort,
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
