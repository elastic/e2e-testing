package main

import (
	"github.com/elastic/metricbeat-tests-poc/cmd"
	"github.com/elastic/metricbeat-tests-poc/config"
	"github.com/elastic/metricbeat-tests-poc/docker"
	"github.com/elastic/metricbeat-tests-poc/services"
)

func init() {
	serviceManager := services.NewServiceManager()

	config.CheckWorkspace(serviceManager.AvailableServices())

	docker.GetDevNetwork()
}

func main() {
	cmd.Execute()
}
