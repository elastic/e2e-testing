package main

import (
	"context"
	log "github.com/sirupsen/logrus"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages-go/v10"
)

func collectsEvents(podName string) error {
	return godog.ErrPending
}

func configurationForHas(podName, option string) error {
	return godog.ErrPending
}

func isDeleted(podName string) error {
	return godog.ErrPending
}

func isDeployed(podName string) error {
	return godog.ErrPending
}

func stopCollectingEvents(podName string) error {
	return godog.ErrPending
}

var cluster kubernetesCluster

func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	suiteContext, cancel := context.WithCancel(context.Background())
	log.DeferExitHandler(cancel)

	ctx.BeforeSuite(func() {
		err := cluster.initialize(suiteContext)
		if err != nil {
			log.WithError(err).Fatal("Failed to initialize cluster")
		}
		log.DeferExitHandler(func() {
			cluster.cleanup(suiteContext)
		})
	})

	ctx.AfterSuite(func() {
		cluster.cleanup(suiteContext)
		cancel()
	})
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	kubectl := cluster.Kubectl()

	scenarioCtx, cancel := context.WithCancel(context.Background())
	log.DeferExitHandler(cancel)

	ctx.BeforeScenario(func(*messages.Pickle) {
		kubectl = kubectl.WithNamespace(scenarioCtx, "")
		if kubectl.Namespace != "" {
			log.Debugf("Running scenario in namespace: %s", kubectl.Namespace)
		}
	})
	ctx.AfterScenario(func(*messages.Pickle, error) {
		kubectl.Cleanup(scenarioCtx)
		cancel()
	})

	ctx.Step(`^a cluster is available$`, func() error { return cluster.isAvailable(scenarioCtx) })
	ctx.Step(`^"([^"]*)" collects events$`, collectsEvents)
	ctx.Step(`^configuration for "([^"]*)" has "([^"]*)"$`, configurationForHas)
	ctx.Step(`^"([^"]*)" is deleted$`, isDeleted)
	ctx.Step(`^"([^"]*)" is deployed$`, isDeployed)
	ctx.Step(`^"([^"]*)" stop collecting events$`, stopCollectingEvents)
}
