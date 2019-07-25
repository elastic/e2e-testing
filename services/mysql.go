package services

// NewMySQLService returns a default MySQL service entity
func NewMySQLService(version string) Service {
	env := map[string]string{
		"MYSQL_ROOT_PASSWORD": "secret",
	}

	return &DockerService{
		ContainerName: "mysql-" + version,
		Daemon:        false,
		Env:           env,
		Image:         "mysql",
		Name:          "mysql",
		Version:       version,
	}
}
