package main

// NewMySQLService returns a default MySQL service entity
func NewMySQLService(version string) Service {
	env := map[string]string{
		"MYSQL_ROOT_PASSWORD": "secret",
	}

	return &DockerService{
		ContainerName: "mysql-" + version,
		Daemon:        false,
		Env:           env,
		ImageTag:      "mysql:" + version,
		Name:          "mysql",
	}
}
