package main

// NewMySQLService returns a default MySQL service entity
func NewMySQLService(daemon bool, version string) Service {
	return Service{
		Daemon: daemon,
		ExposedPorts: []ExposedPort{
			{
				Address:       "0.0.0.0",
				ContainerPort: "3306",
				HostPort:      "3306",
				Protocol:      "tcp",
			},
		},
		ImageTag: "mysql:" + version,
	}
}
