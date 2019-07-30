package services

// NewKafkaService returns a default Kafka service entity
func NewKafkaService(version string, asDaemon bool) Service {
	return &DockerService{
		ContainerName: "kafka-" + version,
		Daemon:        asDaemon,
		ExposedPort:   9092,
		Image:         "wurstmeister/kafka",
		Name:          "kafka",
		Version:       version,
	}
}
