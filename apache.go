package main

// NewApacheService returns a default Apache service entity
func NewApacheService(version string, port string) Service {
	return &DockerService{
		ContainerName: "apache-" + version,
		Daemon:        false,
		ImageTag:      "httpd:" + version,
		Name:          "apache",
	}
}
