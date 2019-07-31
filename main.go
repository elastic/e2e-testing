package main

import (
	"github.com/elastic/metricbeat-tests-poc/cmd"
	"github.com/elastic/metricbeat-tests-poc/config"
	"github.com/elastic/metricbeat-tests-poc/services"
)

func init() {
	serviceManager := services.NewServiceManager()

	config.Init(serviceManager.AvailableServices())
}

func main() {
	cmd.Execute()
}
