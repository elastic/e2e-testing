package main

import "os"

// NewMetricbeatService returns a metricbeat service entity
func NewMetricbeatService(version string, monitoredService Service) Service {
	dir, _ := os.Getwd()

	env := map[string]string{
		"HOST": "localhost",
	}

	bindMounts := map[string]string{
		dir + "/configs/" + monitoredService.GetName() + ".yml": "/usr/share/metricbeat/metricbeat.yml",
		dir + "/outputs": "/tmp",
	}

	labels := map[string]string{
		"co.elastic.logs/module": monitoredService.GetName(),
	}

	return &DockerService{
		Daemon:     false,
		BindMounts: bindMounts,
		Env:        env,
		ImageTag:   "docker.elastic.co/beats/metricbeat:" + version,
		Labels:     labels,
		Name:       "metricbeat",
	}
}
