// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/docker/go-connections/nat"
	apme2e "github.com/elastic/e2e-testing/internal"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/config"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/elasticsearch"
	"github.com/elastic/e2e-testing/internal/installer"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"go.elastic.co/apm"
)

const testResourcesDir = "./testresources"

var fts *FleetTestSuite

var tx *apm.Transaction
var stepSpan *apm.Span

// afterScenario destroys the state created by a scenario
func afterScenario(fts *FleetTestSuite) {
	defer func() {
		fts.DefaultAPIKey = ""
		// Reset Kibana Profile to default
		fts.KibanaProfile = ""
		deployedAgentsCount = 0
	}()

	span := tx.StartSpan("Clean up", "test.scenario.clean", nil)
	fts.currentContext = apm.ContextWithSpan(context.Background(), span)
	defer span.End()

	serviceName := common.ElasticAgentServiceName

	// if DEVELOPER_MODE=true let's not uninstall/unenroll the agent
	if !common.DeveloperMode && fts.InstallerType != "" {
		agentService := deploy.NewServiceRequest(serviceName)

		if !fts.StandAlone {
			// for the centos/debian flavour we need to retrieve the internal log files for the elastic-agent, as they are not
			// exposed as container logs. For that reason we need to go through the installer abstraction
			agentInstaller, _ := installer.Attach(fts.currentContext, fts.getDeployer(), agentService, fts.InstallerType)

			logsPath, _ := filepath.Abs(filepath.Join("..", "..", "..", "outputs", "fleet-", serviceName+uuid.New().String()+".tgz"))
			_, err := shell.Execute(fts.currentContext, ".", "tar", "czf", logsPath, "/opt/Elastic/Agent/data/elastic-agent-*/logs/*")
			if err != nil {
				log.WithFields(log.Fields{
					"serviceName": serviceName,
					"path":        logsPath,
				}).Warn("Failed to collect logs")
			} else {
				log.WithFields(log.Fields{
					"serviceName": serviceName,
					"path":        logsPath,
				}).Info("Logs collected")
			}

			if log.IsLevelEnabled(log.DebugLevel) {
				err := agentInstaller.Logs(fts.currentContext)
				if err != nil {
					log.WithField("error", err).Warn("Could not get agent logs in the container")
				}
			}
			// only call it when the elastic-agent is present
			if !fts.ElasticAgentStopped {
				err := agentInstaller.Uninstall(fts.currentContext)
				if err != nil {
					log.Warnf("Could not uninstall the agent after the scenario: %v", err)
				}
			}
		} else if log.IsLevelEnabled(log.DebugLevel) {
			// for the Docker image, we simply retrieve container logs
			_ = fts.getDeployer().Logs(fts.currentContext, agentService)
		}

		err := fts.unenrollHostname()
		if err != nil {
			manifest, _ := fts.getDeployer().GetServiceManifest(fts.currentContext, agentService)
			log.WithFields(log.Fields{
				"err":      err,
				"hostname": manifest.Hostname,
			}).Warn("The agentIDs for the hostname could not be unenrolled")
		}
	}

	env := fts.getProfileEnv()
	_ = fts.getDeployer().Remove(fts.currentContext, deploy.NewServiceRequest(common.FleetProfileName), []deploy.ServiceRequest{deploy.NewServiceRequest(serviceName)}, env)

	// TODO: Determine why this may be empty here before being cleared out
	if fts.CurrentTokenID != "" {
		err := fts.kibanaClient.DeleteEnrollmentAPIKey(fts.currentContext, fts.CurrentTokenID)
		if err != nil {
			log.WithFields(log.Fields{
				"err":     err,
				"tokenID": fts.CurrentTokenID,
			}).Warn("The enrollment token could not be deleted")
		}
	}

	// TODO: Dont think this is needed if we are making all policies unique
	// fts.kibanaClient.DeleteAllPolicies(fts.currentContext)

	// clean up fields
	fts.CurrentTokenID = ""
	fts.CurrentToken = ""
	fts.InstallerType = ""
	fts.Image = ""
	fts.StandAlone = false
	fts.BeatsProcess = ""
	fts.ElasticAgentFlags = ""
}

// beforeScenario creates the state needed by a scenario
func beforeScenario(fts *FleetTestSuite) {
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute
	exp := utils.GetExponentialBackOff(maxTimeout)

	fts.StandAlone = false
	fts.ElasticAgentStopped = false

	fts.Version = common.ElasticAgentVersion

	waitForPolicy := func() error {
		policy, err := fts.kibanaClient.CreatePolicy(fts.currentContext)
		if err != nil {
			return errors.Wrap(err, "A new policy could not be obtained, retrying.")
		}

		log.WithFields(log.Fields{
			"id":          policy.ID,
			"name":        policy.Name,
			"description": policy.Description,
		}).Info("Policy created")

		fts.Policy = policy

		// Grab the system integration as we'll need to assign it a new name so it wont collide during
		// multiple policy creations at once
		integration, err := fts.kibanaClient.GetIntegrationByPackageName(context.Background(), "system")
		if err != nil {
			return err
		}

		packageDataStream := kibana.PackageDataStream{
			Name:        fmt.Sprintf("%s-%s", integration.Name, uuid.New().String()),
			Description: integration.Title,
			Namespace:   "default",
			PolicyID:    fts.Policy.ID,
			Enabled:     true,
			Package:     integration,
			Inputs:      []kibana.Input{},
		}

		systemMetricsFile := filepath.Join(testResourcesDir, "/default_system_metrics.json")
		jsonData, err := readJSONFile(systemMetricsFile)
		if err != nil {
			return err
		}

		for _, item := range jsonData.Children() {
			var streams []kibana.Stream
			if err := json.Unmarshal(item.Path("streams").Bytes(), &streams); err != nil {
				log.WithError(err).Warn("Could not unmarshall streams, will use an empty array instead")
				streams = []kibana.Stream{}
			}

			if item.Path("type").Data().(string) == "system/metrics" {
				packageDataStream.Inputs = append(packageDataStream.Inputs, kibana.Input{
					Type:    item.Path("type").Data().(string),
					Enabled: item.Path("enabled").Data().(bool),
					Streams: streams,
					Vars: map[string]kibana.Var{
						"system.hostfs": {
							Value: "",
							Type:  "text",
						},
					},
				})
			} else {
				packageDataStream.Inputs = append(packageDataStream.Inputs, kibana.Input{
					Type:    item.Path("type").Data().(string),
					Enabled: item.Path("enabled").Data().(bool),
					Streams: streams,
				})
			}
		}

		err = fts.kibanaClient.AddIntegrationToPolicy(context.Background(), packageDataStream)
		if err != nil {
			return err
		}

		return nil
	}

	err := backoff.Retry(waitForPolicy, exp)
	if err != nil {
		log.Fatal(err)
	}

	// Grab a new enrollment key for new agent
	enrollmentKey, err := fts.kibanaClient.CreateEnrollmentAPIKey(fts.currentContext, fts.Policy)

	if err != nil {
		log.Fatal("Unable to create enrollment token for agent")
	}

	fts.CurrentToken = enrollmentKey.APIKey
	fts.CurrentTokenID = enrollmentKey.ID
}

// bootstrapFleet this method creates the runtime dependencies for the Fleet test suite, being of special
// interest kibana profile passed as part of the environment variables to bootstrap the dependencies.
func bootstrapFleet(ctx context.Context, env map[string]string) error {
	deployer := deploy.New(common.Provider)

	if profile, ok := env["kibanaProfile"]; ok {
		log.Infof("Running kibana with %s profile", profile)
	}

	// the runtime dependencies must be started only in non-remote executions
	return deployer.Bootstrap(ctx, deploy.NewServiceRequest(common.FleetProfileName), env, func() error {
		kibanaClient, err := kibana.NewClient()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"env":   env,
			}).Fatal("Unable to create kibana client")
		}

		err = elasticsearch.WaitForClusterHealth(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Elasticsearch Cluster is not healthy")
		}

		_, err = kibanaClient.WaitForReady(ctx, 10*time.Minute)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"env":   env,
			}).Fatal("Kibana is not healthy")
		}

		err = kibanaClient.RecreateFleet(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"env":   env,
			}).Fatal("Fleet could not be recreated")
		}

		fleetServicePolicy := kibana.FleetServicePolicy

		log.WithFields(log.Fields{
			"id":          fleetServicePolicy.ID,
			"name":        fleetServicePolicy.Name,
			"description": fleetServicePolicy.Description,
		}).Info("Fleet Server Policy retrieved")

		maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
		exp := utils.GetExponentialBackOff(maxTimeout)
		retryCount := 1

		fleetServerBootstrapFn := func() error {
			serviceToken, err := elasticsearch.GetAPIToken(ctx)
			if err != nil {
				retryCount++
				log.WithFields(log.Fields{
					"error":       err,
					"retries":     retryCount,
					"elapsedTime": exp.GetElapsedTime(),
				}).Warn("Could not get API Token from Elasticsearch.")
				return err
			}

			fleetServerEnv := make(map[string]string)
			for k, v := range env {
				fleetServerEnv[k] = v
			}

			fleetServerPort, err := nat.NewPort("tcp", "8220")
			if err != nil {
				retryCount++
				log.WithFields(log.Fields{
					"error":       err,
					"retries":     retryCount,
					"elapsedTime": exp.GetElapsedTime(),
				}).Warn("Could not create TCP port for fleet-server")
				return err
			}

			fleetServerEnv["elasticAgentTag"] = common.ElasticAgentVersion
			fleetServerEnv["fleetServerMode"] = "1"
			fleetServerEnv["fleetServerPort"] = fleetServerPort.Port()
			fleetServerEnv["fleetInsecure"] = "1"
			fleetServerEnv["fleetServerServiceToken"] = serviceToken.AccessToken
			fleetServerEnv["fleetServerPolicyId"] = fleetServicePolicy.ID

			fleetServerSrv := deploy.ServiceRequest{
				Name:    common.ElasticAgentServiceName,
				Flavour: "fleet-server",
			}

			err = deployer.Add(ctx, deploy.NewServiceRequest(common.FleetProfileName), []deploy.ServiceRequest{fleetServerSrv}, fleetServerEnv)
			if err != nil {
				retryCount++
				log.WithFields(log.Fields{
					"error":       err,
					"retries":     retryCount,
					"elapsedTime": exp.GetElapsedTime(),
					"env":         fleetServerEnv,
				}).Warn("Fleet Server could not be started. Retrying")
				return err
			}

			log.WithFields(log.Fields{
				"retries":     retryCount,
				"elapsedTime": exp.GetElapsedTime(),
			}).Info("Fleet Server was started")
			return nil
		}

		err = backoff.Retry(fleetServerBootstrapFn, exp)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Fleet Server could not be started")
		}

		err = kibanaClient.WaitForFleet(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"env":   env,
			}).Fatal("Fleet could not be initialized")
		}
		return nil
	})
}

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

func InitializeFleetTestScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		log.Tracef("Before Fleet scenario: %s", sc.Name)

		tx = apme2e.StartTransaction(sc.Name, "test.scenario")
		tx.Context.SetLabel("suite", "fleet")

		// context is initialised at the step hook, we are initialising it here to prevent panics
		fts.currentContext = context.Background()
		beforeScenario(fts)

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

		afterScenario(fts)

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

	ctx.Step(`^a "([^"]*)" agent is deployed to Fleet$`, fts.anAgentIsDeployedToFleet)
	ctx.Step(`^an agent is deployed to Fleet on top of "([^"]*)"$`, fts.anAgentIsDeployedToFleetOnTopOfBeat)
	ctx.Step(`^an agent is deployed to Fleet with "([^"]*)" installer$`, fts.anAgentIsDeployedToFleetWithInstaller)
	ctx.Step(`^an agent is deployed to Fleet with "([^"]*)" installer and "([^"]*)" flags$`, fts.anAgentIsDeployedToFleetWithInstallerAndTags)
	ctx.Step(`^the agent is listed in Fleet as "([^"]*)"$`, fts.theAgentIsListedInFleetWithStatus)
	ctx.Step(`^the output permissions has "([^"]*)"$`, fts.verifyPermissionHashStatus)
	ctx.Step(`^the host is restarted$`, fts.theHostIsRestarted)
	ctx.Step(`^system package dashboards are listed in Fleet$`, fts.systemPackageDashboardsAreListedInFleet)
	ctx.Step(`^the agent is un-enrolled$`, fts.theAgentIsUnenrolled)
	ctx.Step(`^the agent is re-enrolled on the host$`, fts.theAgentIsReenrolledOnTheHost)
	ctx.Step(`^the enrollment token is revoked$`, fts.theEnrollmentTokenIsRevoked)
	ctx.Step(`^an attempt to enroll a new agent fails$`, fts.anAttemptToEnrollANewAgentFails)
	ctx.Step(`^the "([^"]*)" process is "([^"]*)" on the host$`, fts.processStateChangedOnTheHost)
	ctx.Step(`^the file system Agent folder is empty$`, fts.theFileSystemAgentFolderIsEmpty)
	ctx.Step(`^a Linux data stream exists with some data$`, fts.checkDataStream)

	// preconfigured policies steps
	ctx.Step(`^kibana uses "([^"]*)" profile$`, fts.kibanaUsesProfile)
	ctx.Step(`^agent uses enrollment token from "([^"]*)" policy$`, fts.agentUsesPolicy)
	ctx.Step(`^the agent is enrolled into "([^"]*)" policy$`, fts.agentRunPolicy)

	// upgrade steps
	ctx.Step(`^a "([^"]*)" stale agent is deployed to Fleet with "([^"]*)" installer$`, fts.anStaleAgentIsDeployedToFleetWithInstaller)
	ctx.Step(`^certs are installed$`, fts.installCerts)
	ctx.Step(`^agent is in "([^"]*)" version$`, fts.agentInVersion)
	ctx.Step(`^agent is upgraded to "([^"]*)" version$`, fts.anAgentIsUpgradedToVersion)

	//flags steps
	ctx.Step(`^the elastic agent index contains the tags$`, fts.tagsAreInTheElasticAgentIndex)

	// integrations steps
	ctx.Step(`^the "([^"]*)" integration is "([^"]*)" in the policy$`, fts.theIntegrationIsOperatedInThePolicy)
	ctx.Step(`^the "([^"]*)" datasource is shown in the policy as added$`, fts.thePolicyShowsTheDatasourceAdded)
	ctx.Step(`^an "([^"]*)" is successfully deployed with an Agent using "([^"]*)" installer$`, fts.anIntegrationIsSuccessfullyDeployedWithAgentAndInstaller)

	// endpoint steps
	ctx.Step(`^the host name is shown in the Administration view in the Security App as "([^"]*)"$`, fts.theHostNameIsShownInTheAdminViewInTheSecurityApp)
	ctx.Step(`^the host name is not shown in the Administration view in the Security App$`, fts.theHostNameIsNotShownInTheAdminViewInTheSecurityApp)
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

func InitializeFleetTestSuite(ctx *godog.TestSuiteContext) {
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
		Name:                 "fleet",
		TestSuiteInitializer: InitializeFleetTestSuite,
		ScenarioInitializer:  InitializeFleetTestScenario,
		Options:              &opts,
	}.Run()

	// Optional: Run `testing` package's logic besides godog.
	if st := m.Run(); st > status {
		status = st
	}

	os.Exit(status)
}
