package services

// NewApacheService returns a default Apache service entity
func NewApacheService(version string, asDaemon bool) Service {
	return &DockerService{
		ContainerName: "apache-" + version,
		Daemon:        asDaemon,
		Image:         "httpd",
		Name:          "apache",
		Version:       version,
	}
}
