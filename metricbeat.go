package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	docker "github.com/elastic/metricbeat-tests-poc/docker"
)

// NewMetricbeatService returns a metricbeat service entity
func NewMetricbeatService(version string, monitoredService Service) (Service, error) {
	dir, _ := os.Getwd()

	serviceName := monitoredService.GetName()

	inspect, err := docker.InspectContainer(monitoredService.GetContainerName())
	if err != nil {
		return nil, err
	}

	ip := inspect.NetworkSettings.IPAddress
	fmt.Printf("The monitored service (%s) runs on %s\n", serviceName, ip)
	env := map[string]string{
		"HOST":      ip,
		"FILE_NAME": monitoredService.GetContainerName(),
	}

	bindMounts := map[string]string{
		dir + "/configs/" + serviceName + ".yml": "/usr/share/metricbeat/metricbeat.yml",
		dir + "/outputs":                         "/tmp",
	}

	labels := map[string]string{
		"co.elastic.logs/module": serviceName,
	}

	return &DockerService{
		ContainerName: "metricbeat-" + strconv.Itoa(int(time.Now().UnixNano())),
		Daemon:        false,
		BindMounts:    bindMounts,
		Env:           env,
		ImageTag:      "docker.elastic.co/beats/metricbeat:" + version,
		Labels:        labels,
		Name:          "metricbeat",
	}, nil
}
