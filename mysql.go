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
		ExposedPorts: []ExposedPort{
			{
				Address:       "0.0.0.0",
				ContainerPort: "3306",
				HostPort:      "3306",
				Protocol:      "tcp",
			},
		},
		ImageTag: "mysql:" + version,
		Name:     "mysql",
	}
}
