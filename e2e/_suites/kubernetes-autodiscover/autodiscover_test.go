package main

import (
	"context"
	log "github.com/sirupsen/logrus"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages-go/v10"
)

type podsManager struct {
	kubectl kubernetesControl
	ctx     context.Context
}

func (m *podsManager) collectsEventsFor(observer string, podName string) error {
	return godog.ErrPending
}

func (m *podsManager) configurationForHas(podName, option string) error {
	return godog.ErrPending
}

func (m *podsManager) isDeleted(podName string) error {
	return godog.ErrPending
}

func (m *podsManager) isDeployed(podName string) error {
	return godog.ErrPending
}

func (m *podsManager) stopsCollectingEvents(podName string) error {
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
	scenarioCtx, cancel := context.WithCancel(context.Background())
	log.DeferExitHandler(cancel)

	var kubectl kubernetesControl
	var pods podsManager
	ctx.BeforeScenario(func(*messages.Pickle) {
		kubectl = cluster.Kubectl().WithNamespace(scenarioCtx, "")
		if kubectl.Namespace != "" {
			log.Debugf("Running scenario in namespace: %s", kubectl.Namespace)
		}
		pods.kubectl = kubectl
		pods.ctx = scenarioCtx
	})
	ctx.AfterScenario(func(*messages.Pickle, error) {
		kubectl.Cleanup(scenarioCtx)
		cancel()
	})

	ctx.Step(`^a cluster is available$`, func() error { return cluster.isAvailable(scenarioCtx) })

	ctx.Step(`^configuration for "([^"]*)" has "([^"]*)"$`, pods.configurationForHas)
	ctx.Step(`^"([^"]*)" is deleted$`, pods.isDeleted)
	ctx.Step(`^"([^"]*)" is deployed$`, pods.isDeployed)
	ctx.Step(`^"([^"]*)" collects events for "([^"]*)"$`, pods.collectsEventsFor)
	ctx.Step(`^"([^"]*)" stops collecting events$`, pods.stopsCollectingEvents)
}
