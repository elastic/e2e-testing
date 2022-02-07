// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/cucumber/messages-go/v10"
	"github.com/elastic/e2e-testing/cli/config"
	apme2e "github.com/elastic/e2e-testing/internal"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"go.elastic.co/apm"
)

var imts IngestManagerTestSuite

var tx *apm.Transaction
var stepSpan *apm.Span

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
			kibanaClient:   kibanaClient,
			deployer:       deploy.New(common.Provider),
			dockerDeployer: deploy.New("docker"),
		},
	}
}

func InitializeIngestManagerTestScenario(ctx *godog.ScenarioContext) {
	ctx.BeforeScenario(func(p *messages.Pickle) {
		log.Trace("Before Fleet scenario")

		tx = apme2e.StartTransaction(p.GetName(), "test.scenario")
		tx.Context.SetLabel("suite", "fleet")

		// context is initialised at the step hook, we are initialising it here to prevent panics
		imts.Fleet.currentContext = context.Background()
		imts.Fleet.beforeScenario()
	})

	ctx.AfterScenario(func(p *messages.Pickle, err error) {
		log.Trace("After Fleet scenario")
		if err != nil {
			e := apm.DefaultTracer.NewError(err)
			e.Context.SetLabel("scenario", p.GetName())
			e.Context.SetLabel("gherkin_type", "scenario")
			e.Send()
		}

		f := func() {
			tx.End()

			apm.DefaultTracer.Flush(nil)
		}
		defer f()

		imts.Fleet.afterScenario()
	})

	ctx.BeforeStep(func(step *godog.Step) {
		stepSpan = tx.StartSpan(step.GetText(), "test.scenario.step", nil)
		imts.Fleet.currentContext = apm.ContextWithSpan(context.Background(), stepSpan)
	})
	ctx.AfterStep(func(st *godog.Step, err error) {
		if err != nil {
			e := apm.DefaultTracer.NewError(err)
			e.Context.SetLabel("step", st.GetText())
			e.Context.SetLabel("gherkin_type", "step")
			e.Send()
		}

		if stepSpan != nil {
			stepSpan.End()
		}
	})

	ctx.Step(`^the "([^"]*)" process is in the "([^"]*)" state on the host$`, imts.processStateOnTheHost)
	ctx.Step(`^there are "([^"]*)" instances of the "([^"]*)" process in the "([^"]*)" state$`, imts.thereAreInstancesOfTheProcessInTheState)

	imts.Fleet.contributeSteps(ctx)
}

func InitializeIngestManagerTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		setUpSuite()

		var suiteTx *apm.Transaction
		var suiteParentSpan *apm.Span
		var suiteContext = context.Background()

		// instrumentation
		defer apm.DefaultTracer.Flush(nil)
		suiteTx = apme2e.StartTransaction("Initialise Fleet", "test.suite")
		defer suiteTx.End()
		suiteParentSpan = suiteTx.StartSpan("Before Fleet test suite", "test.suite.before", nil)
		suiteContext = apm.ContextWithSpan(suiteContext, suiteParentSpan)
		defer suiteParentSpan.End()

		deployer := deploy.New(common.Provider)

		err := deployer.PreBootstrap(suiteContext)
		if err != nil {
			log.WithField("error", err).Fatal("Unable to run pre-bootstrap initialization")
		}

		// FIXME: This needs to go into deployer code for docker somehow. Must resolve
		// cyclic imports since common.defaults now imports deploy module
		if !shell.GetEnvBool("SKIP_PULL") && common.Provider != "remote" {
			images := []string{
				"docker.elastic.co/beats/elastic-agent:" + common.BeatVersion,
				"docker.elastic.co/beats/elastic-agent-ubi8:" + common.BeatVersion,
				"docker.elastic.co/elasticsearch/elasticsearch:" + common.StackVersion,
				"docker.elastic.co/observability-ci/elastic-agent:" + common.BeatVersion,
				"docker.elastic.co/observability-ci/elastic-agent-ubi8:" + common.BeatVersion,
				"docker.elastic.co/observability-ci/elasticsearch:" + common.StackVersion,
				"docker.elastic.co/observability-ci/kibana:" + common.KibanaVersion,
			}

			if !strings.HasPrefix(common.KibanaVersion, "pr") {
				images = append(images, "docker.elastic.co/kibana/kibana:"+common.KibanaVersion)
			}

			deploy.PullImages(suiteContext, images)
		}

		common.ProfileEnv = map[string]string{
			"kibanaVersion": common.KibanaVersion,
			"stackPlatform": "linux/" + utils.GetArchitecture(),
			"stackVersion":  common.StackVersion,
		}

		common.ProfileEnv["kibanaProfile"] = "default"
		common.ProfileEnv["kibanaDockerNamespace"] = "kibana"
		if strings.HasPrefix(common.KibanaVersion, "pr") || utils.IsCommit(common.KibanaVersion) {
			// because it comes from a PR
			common.ProfileEnv["kibanaDockerNamespace"] = "observability-ci"
			common.ProfileEnv["KIBANA_IMAGE_REF_CUSTOM"] = "docker.elastic.co/observability-ci/kibana:" + common.KibanaVersion
		}

		if common.Provider != "remote" {
			err := bootstrapFleet(suiteContext, common.ProfileEnv)
			if err != nil {
				log.WithError(err).Fatal("Could not bootstrap Fleet runtime dependencies")
			}
		} else {
			err := imts.Fleet.kibanaClient.WaitForFleet(suiteContext)
			if err != nil {
				log.WithError(err).Fatal("Could not determine Fleet's readiness.")
			}
		}

		imts.Fleet.Version = common.BeatVersionBase
		imts.Fleet.RuntimeDependenciesStartDate = time.Now().UTC()
	})

	ctx.AfterSuite(func() {
		f := func() {
			apm.DefaultTracer.Flush(nil)
		}
		defer f()

		// instrumentation
		var suiteTx *apm.Transaction
		var suiteParentSpan *apm.Span
		var suiteContext = context.Background()
		defer apm.DefaultTracer.Flush(nil)
		suiteTx = apme2e.StartTransaction("Tear Down Fleet", "test.suite")
		defer suiteTx.End()
		suiteParentSpan = suiteTx.StartSpan("After Fleet test suite", "test.suite.after", nil)
		suiteContext = apm.ContextWithSpan(suiteContext, suiteParentSpan)
		defer suiteParentSpan.End()

		if !common.DeveloperMode && common.Provider != "remote" {
			log.Debug("Destroying Fleet runtime dependencies")
			deployer := deploy.New(common.Provider)
			deployer.Destroy(suiteContext, deploy.NewServiceRequest(common.FleetProfileName))
		}
	})
}

var opts = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "progress", // can define default values
}

func init() {
	godog.BindCommandLineFlags("godog.", &opts) // godog v0.11.0 (latest)
}

func TestMain(m *testing.M) {
	flag.Parse()
	opts.Paths = flag.Args()

	status := godog.TestSuite{
		Name:                 "godogs",
		TestSuiteInitializer: InitializeIngestManagerTestSuite,
		ScenarioInitializer:  InitializeIngestManagerTestScenario,
		Options:              &opts,
	}.Run()

	// Optional: Run `testing` package's logic besides godog.
	if st := m.Run(); st > status {
		status = st
	}

	os.Exit(status)
}
