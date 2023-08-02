// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	apme2e "github.com/elastic/e2e-testing/internal"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"go.elastic.co/apm"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/config"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/kubernetes"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/elastic/e2e-testing/pkg/downloads"
)

var beatVersions = map[string]string{}

var defaultEventsWaitTimeout = 300 * time.Second
var defaultDeployWaitTimeout = 300 * time.Second

var tx *apm.Transaction
var stepSpan *apm.Span

type podsManager struct {
	kubectl kubernetes.Control
	ctx     context.Context
}

func (m *podsManager) executeTemplateFor(podName string, writer io.Writer, options []string) error {
	span, _ := apm.StartSpanOptions(m.ctx, "Executing template for pod", "pod.template.execute", apm.SpanOptions{
		Parent: apm.SpanFromContext(m.ctx).TraceContext(),
	})
	span.Context.SetLabel("pod", podName)
	span.Context.SetLabel("options", options)
	defer span.End()

	path := filepath.Join("testdata/templates", sanitizeName(podName)+".yml.tmpl")

	err := m.configureDockerImage(podName)
	if err != nil {
		return err
	}

	usedOptions := make(map[string]bool)
	funcs := template.FuncMap{
		"option": func(o string) bool {
			usedOptions[o] = true
			for _, option := range options {
				if o == option {
					return true
				}
			}
			return false
		},
		"beats_namespace": func() string {
			return deploy.GetDockerNamespaceEnvVarForRepository(podName, "beats")
		},
		"beats_version": func() string {
			return beatVersions[podName]
		},
		"namespace": func() string {
			return m.kubectl.Namespace
		},
		// Can be used to add owner references so cluster-level resources
		// are removed when removing the namespace.
		"namespace_uid": func() string {
			return m.kubectl.NamespaceUID
		},
	}

	t, err := template.New(filepath.Base(path)).Funcs(funcs).ParseFiles(path)
	if os.IsNotExist(err) {
		log.Debugf("template %s does not exist", path)
		return godog.ErrPending
	}
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", path, err)
	}

	err = t.ExecuteTemplate(writer, filepath.Base(path), nil)
	if err != nil {
		return fmt.Errorf("executing template %s: %w", path, err)
	}

	for _, option := range options {
		if _, used := usedOptions[option]; !used {
			log.Debugf("option '%s' is not used in template for '%s'", option, podName)
			return godog.ErrPending
		}
	}

	return nil
}

func (m *podsManager) configureDockerImage(podName string) error {
	namespace := "beats"

	if podName != "filebeat" && podName != "heartbeat" && podName != "metricbeat" && podName != "elastic-agent" && podName != "elasticsearch" {
		log.Debugf("Not processing custom binaries for pod: %s. Only [elasticsearch, filebeat, heartbeat, metricbeat, elastic-agent] will be processed", podName)
		return nil
	}

	span, _ := apm.StartSpanOptions(m.ctx, "Configuring Docker image", "pod.docker-image.configure", apm.SpanOptions{
		Parent: apm.SpanFromContext(m.ctx).TraceContext(),
	})
	span.Context.SetLabel("pod", podName)
	defer span.End()

	// we are caching the versions by pod to avoid downloading and loading/tagging the Docker image multiple times
	if beatVersions[podName] != "" {
		log.Tracef("The beat version was already loaded: %s", beatVersions[podName])
		return nil
	}

	v := common.BeatVersion
	if strings.EqualFold(podName, "elastic-agent") {
		v = common.ElasticAgentVersion
	}
	beatVersion := downloads.GetSnapshotVersion(v) + "-amd64"

	ciSnapshotsFn := downloads.UseBeatsCISnapshots
	if strings.EqualFold(podName, "elastic-agent") {
		ciSnapshotsFn = downloads.UseElasticAgentCISnapshots
	} else if strings.EqualFold(podName, "elasticsearch") {
		// never process elasticsearch artifacts from CI artifacts
		ciSnapshotsFn = func() bool { return false }
	}

	if ciSnapshotsFn() {
		log.Debugf("Configuring Docker image for %s", podName)

		_, imagePath, err := downloads.FetchElasticArtifact(m.ctx, podName, v, "linux", "amd64", "tar.gz", true, true)
		if err != nil {
			return err
		}

		// load the TAR file into the docker host as a Docker image
		err = deploy.LoadImage(imagePath)
		if err != nil {
			return err
		}

		if podName == "elasticsearch" {
			namespace = "elasticsearch"
		}
		err = deploy.TagImage(
			"docker.elastic.co/"+namespace+"/"+podName+":"+downloads.GetSnapshotVersion(common.BeatVersionBase),
			"docker.elastic.co/observability-ci/"+podName+":"+beatVersion,
		)
		if err != nil {
			return err
		}
		// load PR image into kind
		err = cluster.LoadImage(m.ctx, "docker.elastic.co/observability-ci/"+podName+":"+beatVersion)
		if err != nil {
			return err
		}

	}

	log.Tracef("Caching beat version '%s' for %s", beatVersion, podName)
	beatVersions[podName] = beatVersion

	return nil
}

func (m *podsManager) isDeleted(podName string, options []string) error {
	var buf bytes.Buffer
	err := m.executeTemplateFor(podName, &buf, options)
	if err != nil {
		return err
	}

	_, err = m.kubectl.RunWithStdin(m.ctx, &buf, "delete", "-f", "-")
	if err != nil {
		return fmt.Errorf("failed to delete '%s': %w", podName, err)
	}
	return nil
}

func (m *podsManager) isDeployed(podName string, options []string) error {
	var buf bytes.Buffer
	err := m.executeTemplateFor(podName, &buf, options)
	if err != nil {
		return err
	}

	_, err = m.kubectl.RunWithStdin(m.ctx, &buf, "apply", "-f", "-")
	if err != nil {
		return fmt.Errorf("failed to deploy '%s': %w", podName, err)
	}
	return nil
}

func (m *podsManager) isRunning(podName string, options []string) error {
	err := m.isDeployed(podName, options)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(m.ctx, defaultDeployWaitTimeout)
	defer cancel()

	_, err = m.getPodInstances(ctx, podName)
	if err != nil {
		return fmt.Errorf("waiting for instance of '%s': %w", podName, err)
	}
	return nil
}

func (m *podsManager) resourceIs(podName string, state string, options ...string) error {
	span, _ := apm.StartSpanOptions(m.ctx, "Checking resource state", "pod.state.check", apm.SpanOptions{
		Parent: apm.SpanFromContext(m.ctx).TraceContext(),
	})
	span.Context.SetLabel("options", options)
	span.Context.SetLabel("pod", podName)
	span.Context.SetLabel("state", state)
	defer span.End()

	switch state {
	case "running":
		return m.isRunning(podName, options)
	case "deployed":
		return m.isDeployed(podName, options)
	case "deleted":
		return m.isDeleted(podName, options)
	default:
		return godog.ErrPending
	}
}

func (m *podsManager) startEphemeralContainerIn(podName string) error {
	podName = sanitizeName(podName)
	// https://kubernetes.io/docs/tasks/debug/debug-application/debug-running-pod/#ephemeral-container-example
	// example: kubectl debug -it #{podName} -c ephemeral-container --image=busybox:1.28 -- /bin/sh -c "echo Hi from an ephemeral container"
	_, err := m.kubectl.Run(
		m.ctx,
		"debug",
		"-it",
		podName,
		"-c",
		"ephemeral-container",
		"--image=busybox:1.28",
		"--",
		"/bin/sh", "-c",
		"echo Hi from an ephemeral container")
	if err != nil {
		return fmt.Errorf("failed to create ephemeral container: %w. Is EphemeralContainers feature flag enabled in the cluster?", err)
	}
	return nil
}

func (m *podsManager) collectsEventsWith(podName string, condition string) error {
	_, _, ok := splitCondition(condition)
	if !ok {
		return fmt.Errorf("invalid condition '%s'", condition)
	}

	return m.waitForEventsCondition(podName, func(ctx context.Context, localPath string) (bool, error) {
		ok, err := containsEventsWith(m.ctx, localPath, condition)
		if ok {
			return true, nil
		}
		if err != nil {
			log.Debugf("Error checking if %v contains %v: %v", localPath, condition, err)
		}
		return false, nil
	})
}

func (m *podsManager) doesNotCollectEvents(podName, condition, duration string) error {
	_, _, ok := splitCondition(condition)
	if !ok {
		return fmt.Errorf("invalid condition '%s'", condition)
	}

	d, err := time.ParseDuration(duration)
	if err != nil {
		return fmt.Errorf("invalid duration %s: %w", d, err)
	}

	return m.waitForEventsCondition(podName, func(ctx context.Context, localPath string) (bool, error) {
		events, err := readEventsWith(m.ctx, localPath, condition)
		if err != nil {
			return false, err
		}
		// No events ever received, so condition satisfied.
		if len(events) == 0 {
			return true, nil
		}

		lastEvent := events[len(events)-1]
		lastTimestamp, ok := lastEvent["@timestamp"].(string)
		if !ok {
			return false, fmt.Errorf("event %v doesn't contain a @timestamp", lastEvent)
		}
		t, err := time.Parse(time.RFC3339, lastTimestamp)
		if err != nil {
			return false, fmt.Errorf("failed to parse @timestamp %s: %w", lastTimestamp, err)
		}
		if sinceLast := time.Now().Sub(t); sinceLast <= d {
			// Condition cannot be satisfied until the duration has passed after the last
			// event. So wait till then.
			select {
			case <-ctx.Done():
			case <-time.After(d - sinceLast):
			}
			return false, nil
		}

		return true, nil
	})
}

func (m *podsManager) waitForEventsCondition(podName string, conditionFn func(ctx context.Context, localPath string) (bool, error)) error {
	span, _ := apm.StartSpanOptions(m.ctx, "Waiting for events conditions", "pod.events.waitForCondition", apm.SpanOptions{
		Parent: apm.SpanFromContext(m.ctx).TraceContext(),
	})
	span.Context.SetLabel("pod", podName)
	defer span.End()

	ctx, cancel := context.WithTimeout(m.ctx, defaultEventsWaitTimeout)
	defer cancel()

	instances, err := m.getPodInstances(ctx, podName)
	if err != nil {
		return fmt.Errorf("failed to get pod name: %w", err)
	}

	tmpDir, err := ioutil.TempDir(os.TempDir(), "test-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	containerPath := fmt.Sprintf("%s/%s:/tmp/beats-events", m.kubectl.Namespace, instances[0])
	localPath := filepath.Join(tmpDir, "events")
	exp := backoff.WithContext(backoff.NewConstantBackOff(10*time.Second), ctx)
	return backoff.Retry(func() error {
		err := m.copyEvents(ctx, containerPath, localPath)
		if err != nil {
			return fmt.Errorf("failed to copy events from %s: %w", containerPath, err)
		}
		ok, err := conditionFn(ctx, localPath)
		if err != nil {
			return fmt.Errorf("events condition failed: %w", err)
		}
		if !ok {
			return fmt.Errorf("events do not satisfy condition")
		}
		return nil
	}, exp)
}

func (m *podsManager) copyEvents(ctx context.Context, containerPath string, localPath string) error {
	today := time.Now().Format("20060102")
	paths := []string{
		containerPath,

		// Format used since 8.0.
		containerPath + "-" + today + ".ndjson",
	}

	var err error
	var output string
	for _, containerPath := range paths {
		// This command always succeeds, so check if the local path has been created.
		os.Remove(localPath)
		output, _ = m.kubectl.Run(ctx, "cp", "--no-preserve", containerPath, localPath)
		if _, err = os.Stat(localPath); os.IsNotExist(err) {
			continue
		}
		return nil
	}
	log.Debugf("Failed to copy events from %s to %s: %s", containerPath, localPath, output)
	return err
}

func (m *podsManager) getPodInstances(ctx context.Context, podName string) (instances []string, err error) {
	span, _ := apm.StartSpanOptions(m.ctx, "Getting pod instances", "pod.instances.get", apm.SpanOptions{
		Parent: apm.SpanFromContext(m.ctx).TraceContext(),
	})
	span.Context.SetLabel("pod", podName)
	defer span.End()

	app := sanitizeName(podName)
	ticker := backoff.WithContext(backoff.NewConstantBackOff(10*time.Second), ctx)
	err = backoff.Retry(func() error {
		output, err := m.kubectl.Run(ctx, "get", "pods",
			"-l", "k8s-app="+app,
			"--template", `{{range .items}}{{ if eq .status.phase "Running" }}{{.metadata.name}}{{"\n"}}{{ end }}{{end}}`)
		if err != nil {
			return err
		}
		if output == "" {
			return fmt.Errorf("no running pods with label k8s-app=%s found", app)
		}
		instances = strings.Split(strings.TrimSpace(output), "\n")
		return nil
	}, ticker)
	return
}

func splitCondition(c string) (key string, value string, ok bool) {
	fields := strings.SplitN(c, ":", 2)
	if len(fields) != 2 || len(fields[0]) == 0 {
		return
	}

	return fields[0], fields[1], true
}

func flattenMap(m map[string]interface{}) map[string]interface{} {
	flattened := make(map[string]interface{})
	for k, v := range m {
		switch child := v.(type) {
		case map[string]interface{}:
			childMap := flattenMap(child)
			for ck, cv := range childMap {
				flattened[k+"."+ck] = cv
			}
		default:
			flattened[k] = v
		}
	}
	return flattened
}

func containsEventsWith(ctx context.Context, path string, condition string) (bool, error) {
	events, err := readEventsWith(ctx, path, condition)
	if err != nil {
		return false, err
	}
	return len(events) > 0, nil
}

func readEventsWith(ctx context.Context, path string, condition string) ([]map[string]interface{}, error) {
	span, _ := apm.StartSpanOptions(ctx, "Reading events", "kubernetes.events.read", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("condition", condition)
	span.Context.SetLabel("path", path)
	defer span.End()

	key, value, ok := splitCondition(condition)
	if !ok {
		return nil, fmt.Errorf("invalid condition '%s'", condition)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	var events []map[string]interface{}
	decoder := json.NewDecoder(f)
	for decoder.More() {
		var event map[string]interface{}
		err := decoder.Decode(&event)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decoding event: %w", err)
		}

		event = flattenMap(event)
		if v, ok := event[key]; ok && fmt.Sprint(v) == value {
			events = append(events, event)
		}
	}

	return events, nil
}

func sanitizeName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), " ", "-")
}

func waitDuration(ctx context.Context, duration string) error {
	d, err := time.ParseDuration(duration)
	if err != nil {
		return fmt.Errorf("invalid duration %s: %w", d, err)
	}

	select {
	case <-time.After(d):
	case <-ctx.Done():
	}

	return nil
}

var cluster kubernetes.Cluster

func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	suiteContext, cancel := context.WithCancel(context.Background())
	log.DeferExitHandler(cancel)

	ctx.BeforeSuite(func() {
		// init logger
		config.Init()

		common.InitVersions()

		defaultEventsWaitTimeout = defaultEventsWaitTimeout * time.Duration(utils.TimeoutFactor)
		defaultDeployWaitTimeout = defaultDeployWaitTimeout * time.Duration(utils.TimeoutFactor)

		var suiteTx *apm.Transaction
		var suiteParentSpan *apm.Span

		// instrumentation
		defer apm.DefaultTracer.Flush(nil)
		suiteTx = apme2e.StartTransaction("Initialise k8s Autodiscover", "test.suite")
		defer suiteTx.End()
		suiteParentSpan = suiteTx.StartSpan("Before k8s Autodiscover test suite", "test.suite.before", nil)
		suiteContext = apm.ContextWithSpan(suiteContext, suiteParentSpan)
		defer suiteParentSpan.End()

		err := cluster.Initialize(suiteContext, "testdata/kind.yml")
		if err != nil {
			e := apm.DefaultTracer.NewError(err)
			e.Send()

			log.WithError(err).Fatal("Failed to initialize cluster")
		}
		log.DeferExitHandler(func() {
			cluster.Cleanup(suiteContext)
		})
	})

	ctx.AfterSuite(func() {
		f := func() {
			apm.DefaultTracer.Flush(nil)
		}
		defer f()

		// instrumentation
		var suiteTx *apm.Transaction
		var suiteParentSpan *apm.Span
		defer apm.DefaultTracer.Flush(nil)
		suiteTx = apme2e.StartTransaction("Tear Down k8s Autodiscover", "test.suite")
		defer suiteTx.End()
		suiteParentSpan = suiteTx.StartSpan("After k8s Autodiscover test suite", "test.suite.after", nil)
		suiteContext = apm.ContextWithSpan(suiteContext, suiteParentSpan)
		defer suiteParentSpan.End()

		// store cluster logs: see https://kind.sigs.k8s.io/docs/user/quick-start/#exporting-cluster-logs
		clusterName := cluster.Name()
		logsPath, _ := filepath.Abs(filepath.Join("..", "..", "..", "outputs", "kubernetes-autodiscover", clusterName))
		_, err := shell.Execute(suiteContext, ".", "kind", "export", "logs", "--name", clusterName, logsPath)
		if err != nil {
			log.WithFields(log.Fields{
				"cluster": clusterName,
				"path":    logsPath,
			}).Warn("Failed to export Kind cluster logs")
		} else {
			log.WithFields(log.Fields{
				"cluster": clusterName,
				"path":    logsPath,
			}).Info("Kind cluster logs exported")
		}

		if !common.DeveloperMode {
			cluster.Cleanup(suiteContext)
		}
		cancel()
	})
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	scenarioCtx, cancel := context.WithCancel(context.Background())
	log.DeferExitHandler(cancel)

	var kubectl kubernetes.Control
	var pods podsManager
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		tx = apme2e.StartTransaction(sc.Name, "test.scenario")
		tx.Context.SetLabel("suite", "k8s Autodiscover")

		kubectl = cluster.Kubectl().WithNamespace(scenarioCtx, "")
		if kubectl.Namespace != "" {
			log.Debugf("Running scenario %s in namespace: %s", sc.Name, kubectl.Namespace)
		}
		pods.kubectl = kubectl
		pods.ctx = scenarioCtx
		log.DeferExitHandler(func() { kubectl.Cleanup(scenarioCtx) })

		return ctx, nil
	})
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
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

		kubectl.Cleanup(scenarioCtx)
		cancel()

		return ctx, nil
	})

	ctx.StepContext().Before(func(ctx context.Context, step *godog.Step) (context.Context, error) {
		log.Tracef("Before step: %s", step.Text)
		stepSpan = tx.StartSpan(step.Text, "test.scenario.step", nil)
		pods.ctx = apm.ContextWithSpan(scenarioCtx, stepSpan)

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

	ctx.Step(`^"([^"]*)" have passed$`, func(d string) error { return waitDuration(scenarioCtx, d) })

	ctx.Step(`^"([^"]*)" is ([a-z]*)$`, func(name, state string) error {
		return pods.resourceIs(name, state)
	})
	ctx.Step(`^"([^"]*)" is ([a-z]*) with "([^"]*)"$`, func(name, state, option string) error {
		return pods.resourceIs(name, state, option)
	})
	ctx.Step(`^"([^"]*)" is ([a-z]*) with "([^"]*)" and "([^"]*)"$`, func(name, state, option1, option2 string) error {
		return pods.resourceIs(name, state, option1, option2)
	})

	ctx.Step(`^"([^"]*)" collects events with "([^"]*:[^"]*)"$`, pods.collectsEventsWith)
	ctx.Step(`^"([^"]*)" does not collect events with "([^"]*)" during "([^"]*)"$`, pods.doesNotCollectEvents)
	ctx.Step(`^an ephemeral container is started in "([^"]*)"$`, pods.startEphemeralContainerIn)
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
		Name:                 "k8s-autodiscover",
		TestSuiteInitializer: InitializeTestSuite,
		ScenarioInitializer:  InitializeScenario,
		Options:              &opts,
	}.Run()

	// Optional: Run `testing` package's logic besides godog.
	if st := m.Run(); st > status {
		status = st
	}

	os.Exit(status)
}
