package main

import (
	"github.com/elastic/e2e-testing/cli/cmd"
	"github.com/elastic/e2e-testing/cli/config"
)

func init() {
	config.Init()
}

func main() {
	cmd.Execute()
}
