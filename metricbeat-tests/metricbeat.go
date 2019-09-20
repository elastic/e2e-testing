package main

import (
	"context"
	"fmt"
	"os"

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

	indexName := fmt.Sprintf("metricbeat-%s-%s-%s-test", version, serviceName, monitoredService.GetVersion())

	setupCommands := []string{
		"metricbeat",
		"-E", fmt.Sprintf("setup.ilm.rollover_alias=%s", indexName),
		"-E", "output.elasticsearch.hosts=http://elasticsearch:9200",
		"-E", "output.elasticsearch.password=p4ssw0rd",
		"-E", "output.elasticsearch.username=elastic",
		"-E", "setup.kibana.host=http://kibana:5601",
		"-E", "setup.kibana.password=p4ssw0rd",
		"-E", "setup.kibana.username=elastic",
	}

	service.SetBindMounts(bindMounts)
	service.SetCmd(setupCommands)
	service.SetEnv(env)
	service.SetAsDaemon(true)
	service.SetLabels(labels)

	fn := func(ctx context.Context) error {
		return deleteIndex(ctx, "metricbeat", indexName)
	}
	service.SetCleanUp(fn)

	err := serviceManager.Run(service)
	if err != nil {
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
