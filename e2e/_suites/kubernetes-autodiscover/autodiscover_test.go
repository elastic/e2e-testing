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
	"text/template"
	"time"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages-go/v10"
	log "github.com/sirupsen/logrus"

	"github.com/elastic/e2e-testing/cli/shell"
)

const defaultBeatVersion = "8.0.0-SNAPSHOT"
const defaultEventsWaitTimeout = 60 * time.Second

type podsManager struct {
	kubectl kubernetesControl
	ctx     context.Context

	configurations map[string][]string
}

func (m *podsManager) executeTemplateFor(podName string, writer io.Writer, funcmap template.FuncMap) error {
	sanitizedName := strings.ReplaceAll(strings.ToLower(podName), " ", "-")
	path := filepath.Join("testdata/templates", sanitizedName+".yml.tmpl")

	funcs := template.FuncMap{
		"option": func(o string) bool {
			options := m.configurations[podName]
			for _, option := range options {
				if o == option {
					return true
				}
			}
			return false
		},
		"beats_version": func() string {
			return shell.GetEnv("BEAT_VERSION", defaultBeatVersion)
		},
		"namespace": func() string {
			return m.kubectl.Namespace
		},
	}
	for name, f := range funcmap {
		funcs[name] = f
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

	return nil
}

func (m *podsManager) configurationForHas(podName, option string) error {
	used := false
	funcs := template.FuncMap{
		"option": func(o string) bool {
			if o == option {
				used = true
			}
			return true
		},
	}
	err := m.executeTemplateFor(podName, ioutil.Discard, funcs)
	if err != nil {
		return err
	}
	if !used {
		log.Debugf("option '%s' is not used in template for '%s'", option, podName)
		return godog.ErrPending
	}

	if m.configurations == nil {
		m.configurations = make(map[string][]string)
	}
	m.configurations[podName] = append(m.configurations[podName], option)

	return nil
}

func (m *podsManager) isDeleted(podName string) error {
	var buf bytes.Buffer
	err := m.executeTemplateFor(podName, &buf, nil)
	if err != nil {
		return err
	}

	_, err = m.kubectl.RunWithStdin(context.TODO(), &buf, "delete", "-f", "-")
	if err != nil {
		return fmt.Errorf("failed to delete '%s': %w", podName, err)
	}
	return nil
}

func (m *podsManager) isDeployed(podName string) error {
	var buf bytes.Buffer
	err := m.executeTemplateFor(podName, &buf, nil)
	if err != nil {
		return err
	}

	_, err = m.kubectl.RunWithStdin(context.TODO(), &buf, "apply", "-f", "-")
	if err != nil {
		return fmt.Errorf("failed to deploy '%s': %w", podName, err)
	}
	return nil
}

func (m *podsManager) collectsEventsWith(podName string, condition string) error {
	_, _, ok := splitCondition(condition)
	if !ok {
		return fmt.Errorf("invalid condition '%s'", condition)
	}

	tmpDir, err := ioutil.TempDir(os.TempDir(), "test-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx, cancel := context.WithTimeout(context.TODO(), defaultEventsWaitTimeout)
	defer cancel()

	// TODO: Review this, it relies now on the existence of the k8s-app label.
	instance, err := m.getPodInstance(ctx, podName)
	if err != nil {
		return fmt.Errorf("failed to get pod name: %w", err)
	}

	containerPath := fmt.Sprintf("%s/%s:/tmp/beats-events", m.kubectl.Namespace, instance)
	localPath := filepath.Join(tmpDir, "events")
	for {
		_, err := m.kubectl.Run(ctx, "cp", "--no-preserve", containerPath, localPath)
		if err == nil {
			ok, err := containsEventsWith(localPath, condition)
			if ok {
				break
			}
			if err != nil {
				log.Debugf("Error checking if %v contains %v: %v", localPath, condition, err)
			}
		} else {
			log.Debugf("Failed to copy events from %s to %s: %s", containerPath, localPath, err)
		}

		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for events with %s", condition)
		}
	}

	return nil
}

func (m *podsManager) getPodInstance(ctx context.Context, podName string) (string, error) {
	for {
		output, err := m.kubectl.Run(ctx, "get", "pods",
			"-l", "k8s-app="+podName,
			"--template", `{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}`)
		if err != nil {
			return "", err
		}
		if output != "" {
			instances := strings.Split(strings.TrimSpace(output), "\n")
			return instances[0], nil
		}

		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return "", fmt.Errorf("timeout waiting for pod %s", podName)
		}
	}
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

func containsEventsWith(path string, condition string) (bool, error) {
	key, value, ok := splitCondition(condition)
	if !ok {
		return false, fmt.Errorf("invalid condition '%s'", condition)
	}

	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	for decoder.More() {
		var event map[string]interface{}
		err := decoder.Decode(&event)
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, fmt.Errorf("decoding event: %w", err)
		}

		event = flattenMap(event)
		if v, ok := event[key]; ok && fmt.Sprint(v) == value {
			return true, nil
		}
	}

	return false, nil
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
		log.DeferExitHandler(func() { kubectl.Cleanup(scenarioCtx) })
	})
	ctx.AfterScenario(func(*messages.Pickle, error) {
		kubectl.Cleanup(scenarioCtx)
		cancel()
	})

	ctx.Step(`^a cluster is available$`, func() error { return cluster.isAvailable(scenarioCtx) })

	ctx.Step(`^configuration for "([^"]*)" has "([^"]*)"$`, pods.configurationForHas)
	ctx.Step(`^"([^"]*)" is deleted$`, pods.isDeleted)
	ctx.Step(`^"([^"]*)" is deployed$`, pods.isDeployed)
	ctx.Step(`^"([^"]*)" collects events with "([^"]*:[^"]*)"$`, pods.collectsEventsWith)
	ctx.Step(`^"([^"]*)" stops collecting events$`, pods.stopsCollectingEvents)
}
