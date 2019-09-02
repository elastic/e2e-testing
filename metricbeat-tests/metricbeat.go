package main

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/elastic/metricbeat-tests-poc/cli/services"
)

// RunMetricbeatService runs a metricbeat service entity for a service to monitor
func RunMetricbeatService(version string, monitoredService services.Service) (services.Service, error) {
	dir, _ := os.Getwd()

	serviceName := monitoredService.GetName()

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
		"MONITORED_HOST":    monitoredService.GetNetworkAlias(),
	}

	setupCommands := []string{
		"metricbeat",
		"-E", fmt.Sprintf("setup.ilm.rollover_alias=metricbeat-%s-%s-%s", version, serviceName, monitoredService.GetVersion()),
		"-E", "output.elasticsearch.hosts=http://elasticsearch:9200",
		"-E", "output.elasticsearch.password=p4ssw0rd",
		"-E", "output.elasticsearch.username=elastic",
		"-E", "setup.kibana.host=http://kibana:5601",
		"-E", "setup.kibana.password=p4ssw0rd",
		"-E", "setup.kibana.username=elastic",
	}

	service.SetBindMounts(bindMounts)
	service.SetCmd(strings.Join(setupCommands, " "))
	service.SetEnv(env)
	service.SetLabels(labels)

	container, err := service.Run()
	if err != nil || container == nil {
		msg := fmt.Sprintf("Could not run Metricbeat %s for %s %v", version, serviceName, err)

		log.WithFields(log.Fields{
			"error":             err,
			"metricbeatVersion": version,
			"service":           serviceName,
			"serviceVersion":    monitoredService.GetVersion(),
		}).Error(msg)

		return nil, err
	}

	log.WithFields(log.Fields{
		"metricbeatVersion": version,
		"service":           serviceName,
		"serviceVersion":    monitoredService.GetVersion(),
	}).Info("Metricbeat is running configured for the service")

	return service, nil
}
