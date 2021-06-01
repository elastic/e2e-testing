// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"os"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/messages-go/v10"
	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/shell"
	log "github.com/sirupsen/logrus"
)

var imts IngestManagerTestSuite

func setUpSuite() {
	config.Init()

	kibanaClient, err := kibana.NewClient()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	common.InitVersions()

	imts = IngestManagerTestSuite{
		Fleet: &FleetTestSuite{
			kibanaClient: kibanaClient,
			deployer:     deploy.New(common.Provider),
		},
	}
}

func InitializeIngestManagerTestScenario(ctx *godog.ScenarioContext) {
	ctx.BeforeScenario(func(*messages.Pickle) {
		log.Trace("Before Fleet scenario")
		imts.Fleet.beforeScenario()
	})

	ctx.AfterScenario(func(*messages.Pickle, error) {
		log.Trace("After Fleet scenario")
		imts.Fleet.afterScenario()
	})

	ctx.Step(`^the "([^"]*)" process is in the "([^"]*)" state on the host$`, imts.processStateOnTheHost)
	ctx.Step(`^there are "([^"]*)" instances of the "([^"]*)" process in the "([^"]*)" state$`, imts.thereAreInstancesOfTheProcessInTheState)

	imts.Fleet.contributeSteps(ctx)
}

func InitializeIngestManagerTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		setUpSuite()

		log.Trace("Bootstrapping Fleet Server")

		if !shell.GetEnvBool("SKIP_PULL") {
			images := []string{
				"docker.elastic.co/beats/elastic-agent:" + common.BeatVersion,
				"docker.elastic.co/beats/elastic-agent-ubi8:" + common.BeatVersion,
				"docker.elastic.co/elasticsearch/elasticsearch:" + common.StackVersion,
				"docker.elastic.co/kibana/kibana:" + common.KibanaVersion,
				"docker.elastic.co/observability-ci/elastic-agent:" + common.BeatVersion,
				"docker.elastic.co/observability-ci/elastic-agent-ubi8:" + common.BeatVersion,
				"docker.elastic.co/observability-ci/elasticsearch:" + common.StackVersion,
				"docker.elastic.co/observability-ci/elasticsearch-ubi8:" + common.StackVersion,
				"docker.elastic.co/observability-ci/kibana:" + common.KibanaVersion,
				"docker.elastic.co/observability-ci/kibana-ubi8:" + common.KibanaVersion,
			}
			deploy.PullImages(images)
		}

		deployer := deploy.New(common.Provider)
		deployer.Bootstrap(func() error {
			kibanaClient, err := kibana.NewClient()
			if err != nil {
				log.WithField("error", err).Fatal("Unable to create kibana client")
			}
			err = kibanaClient.WaitForFleet()
			if err != nil {
				log.WithField("error", err).Fatal("Fleet could not be initialized")
			}
			return nil
		})

		imts.Fleet.Version = common.BeatVersionBase
		imts.Fleet.RuntimeDependenciesStartDate = time.Now().UTC()
	})

	ctx.AfterSuite(func() {
		if !common.DeveloperMode {
			log.Debug("Destroying Fleet runtime dependencies")
			deployer := deploy.New(common.Provider)
			deployer.Destroy()
		}
	})
}
