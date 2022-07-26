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

var fts *FleetTestSuite

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

	fts = &FleetTestSuite{
		kibanaClient:   kibanaClient,
		deployer:       deploy.New(common.Provider),
		dockerDeployer: deploy.New("docker"),
	}
}

func InitializeIngestManagerTestScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		log.Tracef("Before Fleet scenario: %s", sc.Name)

		tx = apme2e.StartTransaction(sc.Name, "test.scenario")
		tx.Context.SetLabel("suite", "fleet")

		// context is initialised at the step hook, we are initialising it here to prevent panics
		fts.currentContext = context.Background()
		fts.beforeScenario()

		return ctx, nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		log.Tracef("After Fleet scenario: %s", sc.Name)
		if err != nil {
			e := apm.DefaultTracer.NewError(err)
			e.Context.SetLabel("scenario", sc.Name)
			e.Context.SetLabel("gherkin_type", "scenario")
			e.Send()
		}

		f := func() {
			tx.End()

			apm.DefaultTracer.Flush(nil)
		}
		defer f()

		fts.afterScenario()

		log.Tracef("After Fleet scenario: %s", sc.Name)
		return ctx, nil
	})

	ctx.StepContext().Before(func(ctx context.Context, step *godog.Step) (context.Context, error) {
		log.Tracef("Before step: %s", step.Text)
		stepSpan = tx.StartSpan(step.Text, "test.scenario.step", nil)
		fts.currentContext = apm.ContextWithSpan(context.Background(), stepSpan)

		return ctx, nil
	})
	ctx.StepContext().After(func(ctx context.Context, step *godog.Step, status godog.StepResultStatus, err error) (context.Context, error) {
		if err != nil {
			e := apm.DefaultTracer.NewError(err)
			e.Context.SetLabel("step", step.Text)
			e.Context.SetLabel("gherkin_type", "step")
			e.Context.SetLabel("step_status", status.String())
			e.Send()
		}

		if stepSpan != nil {
			stepSpan.End()
		}

		log.Tracef("After step (%s): %s", status.String(), step.Text)
		return ctx, nil
	})

	ctx.Step(`^kibana uses "([^"]*)" profile$`, fts.kibanaUsesProfile)
	ctx.Step(`^agent uses enrollment token from "([^"]*)" policy$`, fts.agentUsesPolicy)
	ctx.Step(`^a "([^"]*)" agent is deployed to Fleet$`, fts.anAgentIsDeployedToFleet)
	ctx.Step(`^an agent is deployed to Fleet on top of "([^"]*)"$`, fts.anAgentIsDeployedToFleetOnTopOfBeat)
	ctx.Step(`^an agent is deployed to Fleet with "([^"]*)" installer$`, fts.anAgentIsDeployedToFleetWithInstaller)
	ctx.Step(`^an agent is deployed to Fleet with "([^"]*)" installer and "([^"]*)" flags$`, fts.anAgentIsDeployedToFleetWithInstallerAndTags)
	ctx.Step(`^a "([^"]*)" stale agent is deployed to Fleet with "([^"]*)" installer$`, fts.anStaleAgentIsDeployedToFleetWithInstaller)
	ctx.Step(`^agent is in "([^"]*)" version$`, fts.agentInVersion)
	ctx.Step(`^agent is upgraded to "([^"]*)" version$`, fts.anAgentIsUpgradedToVersion)
	ctx.Step(`^the agent is listed in Fleet as "([^"]*)"$`, fts.theAgentIsListedInFleetWithStatus)
	ctx.Step(`^the default API key has "([^"]*)"$`, fts.verifyDefaultAPIKey)
	ctx.Step(`^the host is restarted$`, fts.theHostIsRestarted)
	ctx.Step(`^system package dashboards are listed in Fleet$`, fts.systemPackageDashboardsAreListedInFleet)
	ctx.Step(`^the agent is un-enrolled$`, fts.theAgentIsUnenrolled)
	ctx.Step(`^the agent is re-enrolled on the host$`, fts.theAgentIsReenrolledOnTheHost)
	ctx.Step(`^the enrollment token is revoked$`, fts.theEnrollmentTokenIsRevoked)
	ctx.Step(`^an attempt to enroll a new agent fails$`, fts.anAttemptToEnrollANewAgentFails)
	ctx.Step(`^the "([^"]*)" process is "([^"]*)" on the host$`, fts.processStateChangedOnTheHost)
	ctx.Step(`^the file system Agent folder is empty$`, fts.theFileSystemAgentFolderIsEmpty)
	ctx.Step(`^certs are installed$`, fts.installCerts)
	ctx.Step(`^a Linux data stream exists with some data$`, fts.checkDataStream)
	ctx.Step(`^the agent is enrolled into "([^"]*)" policy$`, fts.agentRunPolicy)

	//flags steps
	ctx.Step(`^the elastic agent index contains the tags$`, fts.tagsAreInTheElasticAgentIndex)

	// endpoint steps
	ctx.Step(`^the "([^"]*)" integration is "([^"]*)" in the policy$`, fts.theIntegrationIsOperatedInThePolicy)
	ctx.Step(`^the "([^"]*)" datasource is shown in the policy as added$`, fts.thePolicyShowsTheDatasourceAdded)
	ctx.Step(`^the host name is shown in the Administration view in the Security App as "([^"]*)"$`, fts.theHostNameIsShownInTheAdminViewInTheSecurityApp)
	ctx.Step(`^the host name is not shown in the Administration view in the Security App$`, fts.theHostNameIsNotShownInTheAdminViewInTheSecurityApp)
	ctx.Step(`^an "([^"]*)" is successfully deployed with an Agent using "([^"]*)" installer$`, fts.anIntegrationIsSuccessfullyDeployedWithAgentAndInstaller)
	ctx.Step(`^the policy response will be shown in the Security App$`, fts.thePolicyResponseWillBeShownInTheSecurityApp)
	ctx.Step(`^the policy is updated to have "([^"]*)" in "([^"]*)" mode$`, fts.thePolicyIsUpdatedToHaveMode)
	ctx.Step(`^the policy will reflect the change in the Security App$`, fts.thePolicyWillReflectTheChangeInTheSecurityApp)

	// System Integration steps
	ctx.Step(`^the policy is updated to have "([^"]*)" set to "([^"]*)"$`, fts.thePolicyIsUpdatedToHaveSystemSet)
	ctx.Step(`^"([^"]*)" with "([^"]*)" metrics are present in the datastreams$`, fts.theMetricsInTheDataStream)

	// stand-alone only steps
	ctx.Step(`^a "([^"]*)" stand-alone agent is deployed$`, fts.aStandaloneAgentIsDeployed)
	ctx.Step(`^a "([^"]*)" stand-alone agent is deployed with fleet server mode$`, fts.bootstrapFleetServerFromAStandaloneAgent)
	ctx.Step(`^there is new data in the index from agent$`, fts.thereIsNewDataInTheIndexFromAgent)
	ctx.Step(`^the "([^"]*)" docker container is stopped$`, fts.theDockerContainerIsStopped)
	ctx.Step(`^there is no new data in the index after agent shuts down$`, fts.thereIsNoNewDataInTheIndexAfterAgentShutsDown)
	ctx.Step(`^the stand-alone agent is listed in Fleet as "([^"]*)"$`, fts.theStandaloneAgentIsListedInFleetWithStatus)

	// process steps
	ctx.Step(`^the "([^"]*)" process is in the "([^"]*)" state on the host$`, fts.processStateOnTheHost)
	ctx.Step(`^there are "([^"]*)" instances of the "([^"]*)" process in the "([^"]*)" state$`, fts.thereAreInstancesOfTheProcessInTheState)
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
				"docker.elastic.co/beats/elastic-agent:" + common.ElasticAgentVersion,
				"docker.elastic.co/beats/elastic-agent-ubi8:" + common.ElasticAgentVersion,
				"docker.elastic.co/elasticsearch/elasticsearch:" + common.StackVersion,
				"docker.elastic.co/observability-ci/elastic-agent:" + common.ElasticAgentVersion,
				"docker.elastic.co/observability-ci/elastic-agent-ubi8:" + common.ElasticAgentVersion,
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
			err := fts.kibanaClient.WaitForFleet(suiteContext)
			if err != nil {
				log.WithError(err).Fatal("Could not determine Fleet's readiness.")
			}
		}

		fts.Version = common.BeatVersionBase
		fts.RuntimeDependenciesStartDate = time.Now().UTC()
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
