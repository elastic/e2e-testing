package main

import (
	"fmt"
	"os"
	"path"

	log "github.com/sirupsen/logrus"

	"github.com/elastic/metricbeat-tests-poc/cli/services"
)

// RunMetricbeatService runs a metricbeat service entity for a service to monitor
func RunMetricbeatService(version string, serviceName string, serviceVersion string) error {
	dir, _ := os.Getwd()
	indexName := fmt.Sprintf("metricbeat-%s-%s-%s-test", version, serviceName, serviceVersion)

	serviceManager := services.NewServiceManager()

	env := map[string]string{
		"BEAT_STRICT_PERMS":    "false",
		"indexName":            indexName,
		"metricbeatConfigFile": path.Join(dir, "configurations", serviceName+".yml"),
		"metricbeatTag":        version,
		serviceName + "Tag":    serviceVersion,
		"serviceName":          serviceName,
	}

	err := serviceManager.AddServicesToCompose("metricbeat", []string{"metricbeat"}, env)
	if err != nil {
		msg := fmt.Sprintf("Could not run Metricbeat %s for %s %v", version, serviceName, err)

		log.WithFields(log.Fields{
			"error":             err,
			"metricbeatVersion": version,
			"service":           serviceName,
			"serviceVersion":    serviceVersion,
		}).Error(msg)

		return err
	}

	log.WithFields(log.Fields{
		"metricbeatVersion": version,
		"service":           serviceName,
		"serviceVersion":    serviceVersion,
	}).Info("Metricbeat is running configured for the service")

	// TO-DO: we have to see how to remove the index from compose
	/*
		fn := func(ctx context.Context) error {
			return deleteIndex(ctx, "metricbeat", indexName)
		}
	*/

	return nil
}
