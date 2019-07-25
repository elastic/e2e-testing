package services

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// RunMetricbeatService returns a metricbeat service entity
func RunMetricbeatService(version string, monitoredService Service) (Service, error) {
	dir, _ := os.Getwd()

	serviceName := monitoredService.GetName()

	inspect, err := monitoredService.Inspect()
	if err != nil {
		return nil, err
	}

	ip := inspect.NetworkSettings.IPAddress

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

	fmt.Printf("Metricbeat %s is running configured for %s\n", version, serviceName)

	return service, nil
}
