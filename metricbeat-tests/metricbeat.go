package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/elastic/metricbeat-tests-poc/cli/services"
)

// RunMetricbeatService runs a metricbeat service entity for a service to monitor
func RunMetricbeatService(version string, monitoredService services.Service) (services.Service, error) {
	dir, _ := os.Getwd()

	serviceName := monitoredService.GetName()

	inspect, err := monitoredService.Inspect()
	if err != nil {
		return nil, err
	}

	ip := inspect.NetworkSettings.IPAddress

	bindMounts := map[string]string{
		dir + "/configurations/" + serviceName + ".yml": "/usr/share/metricbeat/metricbeat.yml",
	}

	labels := map[string]string{
		"co.elastic.logs/module": serviceName,
	}

	serviceManager := services.NewServiceManager()

	service := serviceManager.Build("metricbeat", version, false)

	env := map[string]string{
		"BEAT_STRICT_PERMS": "false",
		"HOST":              ip,
		"FILE_NAME":         service.GetName() + "-" + service.GetVersion() + "-" + monitoredService.GetName() + "-" + monitoredService.GetVersion(),
	}

	service.SetBindMounts(bindMounts)
	service.SetEnv(env)
	service.SetLabels(labels)

	container, err := service.Run()
	if err != nil || container == nil {
		return nil, fmt.Errorf("Could not run Metricbeat %s for %s: %v", version, serviceName, err)
	}

	log.WithFields(log.Fields{
		"metricbeatVersion": version,
		"service":           serviceName,
	}).Info("Metricbeat is running configured for the service")

	return service, nil
}
