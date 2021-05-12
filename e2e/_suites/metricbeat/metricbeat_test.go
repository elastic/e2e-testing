// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages-go/v10"
	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/e2e/steps"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/elasticsearch"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// developerMode tears down the backend services (the elasticsearch instance)
// after a test suite. This is the desired behavior, but when developing, we maybe want to keep
// them running to speed up the development cycle.
// It can be overriden by the DEVELOPER_MODE env var
var developerMode = false

var elasticAPMActive = false

<<<<<<< HEAD
var serviceManager deploy.ServiceManager
=======
var serviceManager compose.ServiceManager
>>>>>>> 4c3d3ebe... feat: simplify the initialisation of versions (#1159)

var testSuite MetricbeatTestSuite

var tx *apm.Transaction
var stepSpan *apm.Span

func setupSuite() {
	config.Init()

	developerMode = shell.GetEnvBool("DEVELOPER_MODE")
	if developerMode {
		log.Info("Running in Developer mode 💻: runtime dependencies between different test runs will be reused to speed up dev cycle")
	}

	elasticAPMActive = shell.GetEnvBool("ELASTIC_APM_ACTIVE")
	if elasticAPMActive {
		log.WithFields(log.Fields{
			"apm-environment": shell.GetEnv("ELASTIC_APM_ENVIRONMENT", "local"),
		}).Info("Current execution will be instrumented 🛠")
	}

	common.InitVersions()

<<<<<<< HEAD
	serviceManager = deploy.NewServiceManager()
=======
	serviceManager = compose.NewServiceManager()
>>>>>>> 4c3d3ebe... feat: simplify the initialisation of versions (#1159)

	testSuite = MetricbeatTestSuite{
		Query: elasticsearch.Query{},
	}
}

// MetricbeatTestSuite represents a test suite, holding references to both metricbeat ant
// the service to be monitored
//nolint:unused
type MetricbeatTestSuite struct {
	cleanUpTmpFiles   bool                // if it's needed to clean up temporary files
	configurationFile string              // the  name of the configuration file to be used in this test suite
	ServiceName       string              // the service to be monitored by metricbeat
	ServiceType       string              // the type of the service to be monitored by metricbeat
	ServiceVariant    string              // the variant of the service to be monitored by metricbeat
	ServiceVersion    string              // the version of the service to be monitored by metricbeat
	Query             elasticsearch.Query // the specs for the ES query
	Version           string              // the metricbeat version for the test
	// instrumentation
	currentContext context.Context
}

// getIndexName returns the index to be used when querying Elasticsearch
func (mts *MetricbeatTestSuite) getIndexName() string {
	return mts.Query.IndexName
}

func (mts *MetricbeatTestSuite) setEventModule(eventModule string) {
	mts.Query.EventModule = eventModule
}

// As we are using an index per scenario outline, with an index name formed by metricbeat-version1-module-version2,
// or metricbeat-version1-module-variant-version2,
// and because of the ILM is configured on metricbeat side, then we can use an asterisk for the index name:
// each scenario outline will be namespaced, so no collitions between different test cases should appear
func (mts *MetricbeatTestSuite) setIndexName() {
	mVersion := strings.ReplaceAll(mts.Version, "-SNAPSHOT", "")

	var index string
	if mts.ServiceName != "" {
		if mts.ServiceVariant == "" {
			index = fmt.Sprintf("metricbeat-%s-%s-%s", mVersion, mts.ServiceName, mts.ServiceVersion)
		} else {
			index = fmt.Sprintf("metricbeat-%s-%s-%s-%s", mVersion, mts.ServiceName, mts.ServiceVariant, mts.ServiceVersion)
		}
	} else {
		index = fmt.Sprintf("metricbeat-%s", mVersion)
	}

	index += "-" + utils.RandomString(8)

	mts.Query.IndexName = strings.ToLower(index)
}

func (mts *MetricbeatTestSuite) setServiceVersion(version string) {
	mts.Query.ServiceVersion = version
}

// CleanUp cleans up services in the test suite
func (mts *MetricbeatTestSuite) CleanUp() error {
	span := tx.StartSpan("Clean up", "test.scenario.clean", nil)
	testSuite.currentContext = apm.ContextWithSpan(context.Background(), span)
	defer span.End()

	serviceManager := deploy.NewServiceManager()

	fn := func(ctx context.Context) {
		err := elasticsearch.DeleteIndex(ctx, mts.getIndexName())
		if err != nil {
			log.WithFields(log.Fields{
				"profile": "metricbeat",
				"index":   mts.getIndexName(),
			}).Warn("The index was not deleted, but we are not failing the test case")
		}
	}
	defer fn(context.Background())

	env := map[string]string{
		"stackVersion": common.StackVersion,
	}

	services := []deploy.ServiceRequest{deploy.NewServiceRequest("metricbeat")}
	if mts.ServiceName != "" {
		services = append(services, deploy.NewServiceRequest(mts.ServiceName))
	}

	err := serviceManager.RemoveServicesFromCompose(mts.currentContext, deploy.NewServiceRequest("metricbeat"), services, env)

	if mts.cleanUpTmpFiles {
		if _, err := os.Stat(mts.configurationFile); err == nil {
			beatsLocalPath := shell.GetEnv("BEATS_LOCAL_PATH", "")
			if beatsLocalPath == "" {
				os.Remove(mts.configurationFile)

				log.WithFields(log.Fields{
					"path": mts.configurationFile,
				}).Trace("Metricbeat configuration file removed.")
			} else {
				log.WithFields(log.Fields{
					"path": mts.configurationFile,
				}).Trace("Metricbeat configuration file not removed because it's part of a repository.")
			}
		}
	}

	return err
}

func InitializeMetricbeatScenarios(ctx *godog.ScenarioContext) {
	ctx.BeforeScenario(func(p *messages.Pickle) {
		log.Trace("Before Metricbeat scenario...")

		tx = apm.DefaultTracer.StartTransaction(p.GetName(), "test.scenario")
		tx.Context.SetLabel("suite", "metricbeat")
	})

	ctx.AfterScenario(func(*messages.Pickle, error) {
		f := func() {
			tx.End()

			apm.DefaultTracer.Flush(nil)
		}
		defer f()

		log.Trace("After Metricbeat scenario...")
		err := testSuite.CleanUp()
		if err != nil {
			log.Errorf("CleanUp failed: %v", err)
		}
	})

	ctx.BeforeStep(func(step *godog.Step) {
		stepSpan = tx.StartSpan(step.GetText(), "test.scenario.step", nil)
		testSuite.currentContext = apm.ContextWithSpan(context.Background(), stepSpan)
	})
	ctx.AfterStep(func(st *godog.Step, err error) {
		if stepSpan != nil {
			stepSpan.End()
		}
	})

	ctx.Step(`^"([^"]*)" "([^"]*)" is running for metricbeat$`, testSuite.serviceIsRunningForMetricbeat)
	ctx.Step(`^"([^"]*)" v([^"]*), variant of "([^"]*)", is running for metricbeat$`, testSuite.serviceVariantIsRunningForMetricbeat)
	ctx.Step(`^metricbeat is installed and configured for "([^"]*)" module$`, testSuite.installedAndConfiguredForModule)
	ctx.Step(`^metricbeat is installed and configured for "([^"]*)", variant of the "([^"]*)" module$`, testSuite.installedAndConfiguredForVariantModule)
	ctx.Step(`^there are no errors in the index$`, testSuite.thereAreNoErrorsInTheIndex)
	ctx.Step(`^there are "([^"]*)" events in the index$`, testSuite.thereAreEventsInTheIndex)

	ctx.Step(`^metricbeat is installed using "([^"]*)" configuration$`, testSuite.installedUsingConfiguration)
}

// InitializeMetricbeatTestSuite adds steps to the Godog test suite
//nolint:deadcode,unused
func InitializeMetricbeatTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		setupSuite()
		log.Trace("Before Metricbeat Suite...")

		var suiteTx *apm.Transaction
		var suiteParentSpan *apm.Span
		var suiteContext = context.Background()

		// instrumentation
		defer apm.DefaultTracer.Flush(nil)
		suiteTx = apm.DefaultTracer.StartTransaction("Initialise Metricbeat", "test.suite")
		defer suiteTx.End()
		suiteParentSpan = suiteTx.StartSpan("Before Metricbeat test suite", "test.suite.before", nil)
		suiteContext = apm.ContextWithSpan(suiteContext, suiteParentSpan)
		defer suiteParentSpan.End()

		serviceManager := deploy.NewServiceManager()

		env := map[string]string{
			"stackVersion": common.StackVersion,
		}

		err := serviceManager.RunCompose(
			suiteContext, true, []deploy.ServiceRequest{deploy.NewServiceRequest("metricbeat")}, env)
		if err != nil {
			log.WithFields(log.Fields{
				"profile": "metricbeat",
			}).Fatal("Could not run the profile.")
		}

		minutesToBeHealthy := time.Duration(utils.TimeoutFactor) * time.Minute
		healthy, err := elasticsearch.WaitForElasticsearch(suiteContext, minutesToBeHealthy)
		if !healthy {
			log.WithFields(log.Fields{
				"error":   err,
				"minutes": minutesToBeHealthy,
			}).Fatal("The Elasticsearch cluster could not get the healthy status")
		}

		elasticAPMEnvironment := shell.GetEnv("ELASTIC_APM_ENVIRONMENT", "ci")
		if elasticAPMActive && elasticAPMEnvironment == "local" {
			steps.AddAPMServicesForInstrumentation(suiteContext, "metricbeat", common.StackVersion, true, env)
		}
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
		suiteTx = apm.DefaultTracer.StartTransaction("Tear Down Metricbeat", "test.suite")
		defer suiteTx.End()
		suiteParentSpan = suiteTx.StartSpan("After Metricbeat test suite", "test.suite.after", nil)
		suiteContext = apm.ContextWithSpan(suiteContext, suiteParentSpan)
		defer suiteParentSpan.End()

		if !developerMode {
			serviceManager := deploy.NewServiceManager()
			err := serviceManager.StopCompose(suiteContext, true, []deploy.ServiceRequest{deploy.NewServiceRequest("metricbeat")})
			if err != nil {
				log.WithFields(log.Fields{
					"profile": "metricbeat",
				}).Error("Could not stop the profile.")
			}
		}
	})
}

func (mts *MetricbeatTestSuite) installedAndConfiguredForModule(serviceType string) error {
	serviceType = strings.ToLower(serviceType)

	// at this point we have everything to define the index name
	mts.Version = common.BeatVersion
	mts.setIndexName()
	mts.ServiceType = serviceType

	// look up configurations under workspace's configurations directory. If does not exist, look up tool's workspace
	dir, _ := os.Getwd()
	configFile := path.Join(dir, "configurations", mts.ServiceName+".yml")
	found, err := config.FileExists(configFile)
	if !found || err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"path":    configFile,
			"service": mts.ServiceName,
		}).Debug("Could not retrieve configuration file under test workspace. Looking up tool's workspace")

		configFile = path.Join(config.Op.Workspace, "compose", "services", mts.ServiceName, "_meta", "config.yml")
		ok, err := config.FileExists(configFile)
		if !ok {
			return fmt.Errorf("The configuration file for %s does not exist", mts.ServiceName)
		}
		if err != nil {
			return err
		}
	}

	mts.configurationFile = configFile

	_ = os.Chmod(mts.configurationFile, 0666)

	mts.setEventModule(mts.ServiceType)
	mts.setServiceVersion(mts.Version)

	err = mts.runMetricbeatService()
	if err != nil {
		return err
	}

	return nil
}

func (mts *MetricbeatTestSuite) installedAndConfiguredForVariantModule(serviceVariant string, serviceType string) error {
	mts.ServiceVariant = serviceVariant

	err := mts.installedAndConfiguredForModule(serviceType)

	if err != nil {
		return err
	}

	return nil
}

func (mts *MetricbeatTestSuite) installedUsingConfiguration(configuration string) error {
	// at this point we have everything to define the index name
	mts.Version = common.BeatVersion
	mts.setIndexName()

	configurationFilePath, err := steps.FetchBeatConfiguration(false, "metricbeat", configuration+".yml")
	if err != nil {
		return err
	}

	mts.configurationFile = configurationFilePath
	mts.cleanUpTmpFiles = true

	mts.setEventModule("system")
	mts.setServiceVersion(mts.Version)

	err = mts.runMetricbeatService()
	if err != nil {
		return err
	}

	return nil
}

// runMetricbeatService runs a metricbeat service entity for a service to monitor it
func (mts *MetricbeatTestSuite) runMetricbeatService() error {
	useCISnapshots := shell.GetEnvBool("BEATS_USE_CI_SNAPSHOTS")
	beatsLocalPath := shell.GetEnv("BEATS_LOCAL_PATH", "")
	if useCISnapshots || beatsLocalPath != "" {
		artifactName := utils.BuildArtifactName("metricbeat", mts.Version, common.BeatVersionBase, "linux", "amd64", "tar.gz", true)

		imagePath, err := utils.FetchBeatsBinary(artifactName, "metricbeat", mts.Version, common.BeatVersionBase, utils.TimeoutFactor, true)
		if err != nil {
			return err
		}

		err = deploy.LoadImage(imagePath)
		if err != nil {
			return err
		}

		mts.Version = mts.Version + "-amd64"

<<<<<<< HEAD
		err = deploy.TagImage(
=======
		err = docker.TagImage(
>>>>>>> 4c3d3ebe... feat: simplify the initialisation of versions (#1159)
			"docker.elastic.co/beats/metricbeat:"+common.BeatVersionBase,
			"docker.elastic.co/observability-ci/metricbeat:"+mts.Version,
		)
		if err != nil {
			return err
		}
	}

	// this is needed because, in general, the target service (apache, mysql, redis) does not have a healthcheck
	waitForService := time.Duration(utils.TimeoutFactor) * 10 * time.Second
	if mts.ServiceName == "ceph" {
		// see https://github.com/elastic/beats/blob/ef6274d0d1e36308a333cbed69846a1bd63528ae/metricbeat/module/ceph/mgr_osd_tree/mgr_osd_tree_integration_test.go#L35
		// Ceph service needs more time to start up
		waitForService = waitForService * 4
	}

	utils.Sleep(waitForService)

	serviceManager := deploy.NewServiceManager()

	logLevel := log.GetLevel().String()
	if log.GetLevel() == log.TraceLevel {
		logLevel = log.DebugLevel.String()
	}

	env := map[string]string{
		"BEAT_STRICT_PERMS":     "false",
		"indexName":             mts.getIndexName(),
		"logLevel":              logLevel,
		"metricbeatConfigFile":  mts.configurationFile,
		"metricbeatTag":         mts.Version,
		"stackVersion":          common.StackVersion,
		mts.ServiceName + "Tag": mts.ServiceVersion,
		"serviceName":           mts.ServiceName,
	}

	env["metricbeatDockerNamespace"] = utils.GetDockerNamespaceEnvVar("beats")
	env["metricbeatPlatform"] = "linux/amd64"

	err := serviceManager.AddServicesToCompose(
		testSuite.currentContext,
		deploy.NewServiceRequest("metricbeat"),
		[]deploy.ServiceRequest{deploy.NewServiceRequest("metricbeat")},
		env)
	if err != nil {
		log.WithFields(log.Fields{
			"error":             err,
			"metricbeatVersion": mts.Version,
			"service":           mts.ServiceName,
			"serviceVersion":    mts.ServiceVersion,
		}).Error("Could not run Metricbeat for the service")

		return err
	}

	if mts.ServiceName != "" && mts.ServiceVersion != "" {
		fields := log.Fields{
			"metricbeatVersion": mts.Version,
			"service":           mts.ServiceName,
			"serviceVersion":    mts.ServiceVersion,
		}

		if mts.ServiceVariant != "" {
			fields["variant"] = mts.ServiceVariant
		}

		log.WithFields(fields).Info("Metricbeat is running configured for the service")
	} else {
		log.WithFields(log.Fields{
			"metricbeatVersion": mts.Version,
		}).Info("Metricbeat is running")
	}

	if log.IsLevelEnabled(log.DebugLevel) {
		services := []deploy.ServiceRequest{
			deploy.NewServiceRequest("metricbeat"), // profile name
			deploy.NewServiceRequest("metricbeat"), // metricbeat service
		}

		if developerMode {
			err = serviceManager.RunCommand(deploy.NewServiceRequest("metricbeat"), services, []string{"logs", "metricbeat"}, env)
			if err != nil {
				log.WithFields(log.Fields{
					"error":             err,
					"metricbeatVersion": mts.Version,
					"service":           mts.ServiceName,
					"serviceVersion":    mts.ServiceVersion,
				}).Warn("Could not retrieve Metricbeat logs")
			}
		}
	}

	return nil
}

func (mts *MetricbeatTestSuite) serviceIsRunningForMetricbeat(serviceType string, serviceVersion string) error {
	serviceType = strings.ToLower(serviceType)

	env := map[string]string{
		"stackVersion": common.StackVersion,
	}
	env = config.PutServiceEnvironment(env, serviceType, serviceVersion)

	err := serviceManager.AddServicesToCompose(
		testSuite.currentContext, deploy.NewServiceRequest("metricbeat"),
		[]deploy.ServiceRequest{deploy.NewServiceRequest(serviceType)},
		env)
	if err != nil {
		log.WithFields(log.Fields{
			"service": serviceType,
			"version": serviceVersion,
		}).Error("Could not run the service.")
	}

	mts.ServiceName = serviceType
	mts.ServiceVersion = serviceVersion

	return err
}

func (mts *MetricbeatTestSuite) serviceVariantIsRunningForMetricbeat(
	serviceVariant string, serviceVersion string, serviceType string) error {

	serviceVariant = strings.ToLower(serviceVariant)
	serviceType = strings.ToLower(serviceType)

	env := map[string]string{
		"stackVersion": common.StackVersion,
	}
	env = config.PutServiceVariantEnvironment(env, serviceType, serviceVariant, serviceVersion)

	err := serviceManager.AddServicesToCompose(
		testSuite.currentContext, deploy.NewServiceRequest("metricbeat"),
		[]deploy.ServiceRequest{deploy.NewServiceRequest(serviceType)},
		env)
	if err != nil {
		log.WithFields(log.Fields{
			"service": serviceType,
			"variant": serviceVariant,
			"version": serviceVersion,
		}).Error("Could not run the service.")
	}

	mts.ServiceName = serviceType
	mts.ServiceVersion = serviceVersion

	return err
}

func (mts *MetricbeatTestSuite) thereAreEventsInTheIndex() error {
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"event.module": mts.Query.EventModule,
						},
					},
				},
			},
		},
	}

	minimumHitsCount := 5
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute

	result, err := elasticsearch.WaitForNumberOfHits(mts.currentContext, mts.getIndexName(), esQuery, minimumHitsCount, maxTimeout)
	if err != nil {
		return err
	}

	err = elasticsearch.AssertHitsArePresent(result)
	if err != nil {
		log.WithFields(log.Fields{
			"eventModule":    mts.Query.EventModule,
			"index":          mts.Query.IndexName,
			"serviceVersion": mts.Query.ServiceVersion,
		}).Error(err.Error())
		return err
	}

	return nil
}

func (mts *MetricbeatTestSuite) thereAreNoErrorsInTheIndex() error {
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"event.module": mts.Query.EventModule,
						},
					},
				},
			},
		},
	}

	minimumHitsCount := 5
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute

	result, err := elasticsearch.WaitForNumberOfHits(mts.currentContext, mts.getIndexName(), esQuery, minimumHitsCount, maxTimeout)
	if err != nil {
		return err
	}

	return elasticsearch.AssertHitsDoNotContainErrors(result, mts.Query)
}
