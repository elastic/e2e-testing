package services

// NewApacheService returns a default Apache service entity
func NewApacheService(version string, asDaemon bool) Service {
	return &DockerService{
		ContainerName: "apache-" + version,
		Daemon:        asDaemon,
		ExposedPorts: []ExposedPort{
			{
				Address:       "0.0.0.0",
				ContainerPort: "80",
				Protocol:      "tcp",
			},
		},
		Image:   "httpd",
		Name:    "apache",
		Version: version,
	}
}
