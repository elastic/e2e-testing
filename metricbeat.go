package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	docker "github.com/elastic/metricbeat-tests-poc/docker"
)

// RunMetricbeatService returns a metricbeat service entity
func RunMetricbeatService(version string, monitoredService Service) (Service, error) {
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

	service := &DockerService{
		ContainerName: "metricbeat-" + strconv.Itoa(int(time.Now().UnixNano())),
		Daemon:        false,
		BindMounts:    bindMounts,
		Env:           env,
		Image:         "docker.elastic.co/beats/metricbeat",
		Labels:        labels,
		Name:          "metricbeat",
		Version:       version,
	}

	if service == nil {
		return nil, fmt.Errorf("Could not create Metricbeat %s service for %s", version, serviceName)
	}

	container, err := service.Run()
	if err != nil || container == nil {
		return nil, fmt.Errorf("Could not run Metricbeat %s: %v", version, err)
	}

	ctx := context.Background()

	ip, err1 := container.Host(ctx)
	if err1 != nil {
		return nil, fmt.Errorf("Could not get Metricbeat %s host: %v", version, err1)
	}

	fmt.Printf(
		"Metricbeat %s is running configured for %s on IP %s\n", version, serviceName, ip)

	return service, nil
}
