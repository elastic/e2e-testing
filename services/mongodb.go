package services

// NewMongoDBService returns a default Apache service entity
func NewMongoDBService(version string, asDaemon bool) Service {
	return &DockerService{
		ContainerName: "mongodb-" + version,
		Daemon:        asDaemon,
		ExposedPort:   27017,
		Image:         "mongo",
		Name:          "mongodb",
		Version:       version,
	}
}
