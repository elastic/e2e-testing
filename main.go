package main

import "github.com/elastic/metricbeat-tests-poc/cmd"
import "github.com/elastic/metricbeat-tests-poc/docker"

func init() {
	docker.GetDevNetwork()
}

func main() {
	cmd.Execute()
}
