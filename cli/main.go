package main

import (
	"github.com/elastic/metricbeat-tests-poc/cli/cmd"
	"github.com/elastic/metricbeat-tests-poc/cli/config"
)

func init() {
	config.Init()
}

func main() {
	cmd.Execute()
}
