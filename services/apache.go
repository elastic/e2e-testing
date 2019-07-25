package services

// NewApacheService returns a default Apache service entity
func NewApacheService(version string) Service {
	return &DockerService{
		ContainerName: "apache-" + version,
		Daemon:        false,
		Image:         "httpd",
		Name:          "apache",
		Version:       version,
	}
}
