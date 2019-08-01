package main

import (
	"github.com/elastic/metricbeat-tests-poc/cmd"
	"github.com/elastic/metricbeat-tests-poc/config"
)

func init() {
	config.Init()
}

func main() {
	cmd.Execute()
}
