package services

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/elastic/metricbeat-tests-poc/log"
)

// NewMetricbeatService returns a metricbeat service entity
func NewMetricbeatService(version string, asDaemon bool) Service {
	service := &DockerService{
		ContainerName: "metricbeat-" + version + "-" + strconv.Itoa(int(time.Now().UnixNano())),
		Daemon:        asDaemon,
		Image:         "docker.elastic.co/beats/metricbeat",
		Name:          "metricbeat",
		Version:       version,
	}

	return service
}

// RunMetricbeatService runs a metricbeat service entity for a service to monitor
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

	service := NewMetricbeatService(version, false)

	service.SetBindMounts(bindMounts)
	service.SetEnv(env)
	service.SetLabels(labels)

	container, err := service.Run()
	if err != nil || container == nil {
		return nil, fmt.Errorf("Could not run Metricbeat %s for %s: %v", version, serviceName, err)
	}

	log.Info("Metricbeat %s is running configured for %s", version, serviceName)

	return service, nil
}
