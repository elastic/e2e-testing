package main

import (
	"flag"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"os"
	"testing"
)

var opts = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "progress",
}

func init() {
	godog.BindCommandLineFlags("godog.", &opts)
}

func InitializeIngestManagerTestSuite(ctx *godog.TestSuiteContext) {
	initializeIngestManagerTestSuite(ctx)
}

func InitializeIngestManagerTestScenario(ctx *godog.ScenarioContext) {
	initializeIngestManagerTestScenario(ctx)
}

func TestMain(m *testing.M) {
	flag.Parse()
	opts.Paths = flag.Args()

	status := godog.TestSuite{
		Name:                 "fleet",
		TestSuiteInitializer: InitializeIngestManagerTestSuite,
		ScenarioInitializer:  InitializeIngestManagerTestScenario,
		Options:              &opts,
	}.Run()

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}
