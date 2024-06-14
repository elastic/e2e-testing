// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetes

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm/v2"

	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
)

// Control struct for k8s cluster
type Control struct {
	config           string
	Namespace        string
	NamespaceUID     string
	createdNamespace bool
}

// WithConfig config setter
func (c Control) WithConfig(config string) Control {
	c.config = config
	return c
}

// WithNamespace namespace setter
func (c Control) WithNamespace(ctx context.Context, namespace string) Control {
	if namespace == "" {
		namespace = "test-" + uuid.New().String()
		err := c.createNamespace(ctx, namespace)
		if err != nil {
			log.WithError(err).Fatalf("Failed to create namespace %s", namespace)
		}
		c.createdNamespace = true
	}
	uid, err := c.Run(ctx, "get", "namespace", namespace, "-o", "jsonpath={.metadata.uid}")
	if err != nil {
		log.WithError(err).Fatalf("Failed to get namespace %s uid", namespace)
	}
	c.NamespaceUID = uid
	c.Namespace = namespace
	return c
}

func (c Control) createNamespace(ctx context.Context, namespace string) error {
	if namespace == "" {
		return nil
	}

	_, err := c.Run(ctx, "create", "namespace", namespace)
	if err != nil {
		return fmt.Errorf("namespace creation failed: %w", err)
	}

	// Wait for default account to be available, if not it is not possible to
	// deploy pods in this namespace.
	timeout := 60 * time.Second
	exp := backoff.WithContext(utils.GetExponentialBackOff(timeout), ctx)
	return backoff.Retry(func() error {
		_, err := c.Run(ctx, "get", "serviceaccount", "default")
		if err != nil {
			return fmt.Errorf("namespace was created but still not ready: %w", err)
		}
		return nil
	}, exp)
}

// Cleanup deletes k8s namespace
func (c Control) Cleanup(ctx context.Context) error {
	if c.createdNamespace && c.Namespace != "" {
		output, err := c.Run(ctx, "delete", "namespace", c.Namespace)
		if err != nil {
			return fmt.Errorf("failed to delete namespace %s: %v: %s", c.Namespace, err, output)
		}
	}
	return nil
}

// Run ability to run kubectl commands
func (c Control) Run(ctx context.Context, runArgs ...string) (output string, err error) {
	return c.RunWithStdin(ctx, nil, runArgs...)
}

// RunWithStdin run kubectl commands passing in options from stdin
func (c Control) RunWithStdin(ctx context.Context, stdin io.Reader, runArgs ...string) (output string, err error) {
	shell.CheckInstalledSoftware("kubectl")
	var args []string
	if c.config != "" {
		args = append(args, "--kubeconfig", c.config)
	}
	if c.Namespace != "" {
		args = append(args, "--namespace", c.Namespace)
	}
	args = append(args, runArgs...)
	return shell.ExecuteWithStdin(ctx, ".", stdin, "kubectl", map[string]string{}, args...)
}

// Cluster kind structure definition
type Cluster struct {
	kindName   string
	kubeconfig string

	tmpDir string
}

// Kubectl executable reference to kubectl with applied kubeconfig
func (c Cluster) Kubectl() Control {
	return Control{}.WithConfig(c.kubeconfig)
}

// Name returns cluster name
func (c Cluster) Name() string {
	return c.kindName
}

func (c Cluster) isAvailable(ctx context.Context) error {
	out, err := c.Kubectl().Run(ctx, "api-versions")
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(out)) == 0 {
		return fmt.Errorf("no api versions?")
	}

	return nil
}

// Initialize detect existing cluster contexts, otherwise will create one via Kind
func (c *Cluster) Initialize(ctx context.Context, kindConfigPath string) error {
	span, _ := apm.StartSpanOptions(ctx, "Initialising kubernetes cluster", "kind.cluster.initialize", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	err := c.isAvailable(ctx)
	if err == nil {
		return nil
	}

	log.Info("Kubernetes cluster not available, will start one using kind")
	shell.CheckInstalledSoftware("kind")
	kindVersion, err := shell.Execute(ctx, ".", "kind", "version")
	if err != nil {
		log.WithError(err).Fatal("Failed to get kind version")
	}
	log.Infof("Using %s", kindVersion)

	c.tmpDir, err = os.MkdirTemp(os.TempDir(), "test-")
	if err != nil {
		log.WithError(err).Fatal("Failed to create temporary directory")
	}

	name := "kind-" + uuid.New().String()
	c.kubeconfig = filepath.Join(c.tmpDir, "kubeconfig")

	args := []string{
		"create", "cluster",
		"--name", name,
		"--config", kindConfigPath,
		"--kubeconfig", c.kubeconfig,
	}
	span.Context.SetLabel("arguments", args)

	if version, ok := os.LookupEnv("KUBERNETES_VERSION"); ok && version != "" {
		log.Infof("Installing Kubernetes v%s", version)
		args = append(args, "--image", "kindest/node:v"+version)
	}
	output, err := shell.Execute(ctx, ".", "kind", args...)
	if err != nil {
		log.WithError(err).WithField("output", output).Fatal("Failed to create kind cluster")
		return err
	}
	c.kindName = name

	log.Infof("Kubeconfig in %s", c.kubeconfig)

	return nil
}

// Cleanup deletes the kind cluster if available
func (c *Cluster) Cleanup(ctx context.Context) {
	span, _ := apm.StartSpanOptions(ctx, "Cleanup cluster", "kind.cluster.cleanup", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	if c.kindName != "" {
		_, err := shell.Execute(ctx, ".", "kind", "delete", "cluster", "--name", c.kindName)
		if err != nil {
			log.Warnf("Failed to delete kind cluster %s", c.kindName)
		}
		c.kindName = ""
		log.Infof("kind cluster %s was deleted", c.kindName)
	}
	if c.tmpDir != "" {
		err := os.RemoveAll(c.tmpDir)
		if err != nil {
			log.Warnf("Failed to remove temporary directory %s", c.tmpDir)
		}
	}
}

// LoadImage loads a Docker image into Kind runtime, using it fully qualified name.
// It does not check cluster availability because a pull error could be present in the pod,
// which will need the load of the requested image, causing a chicken-egg error.
func (c *Cluster) LoadImage(ctx context.Context, image string) error {
	span, _ := apm.StartSpanOptions(ctx, "Loading image into cluster", "kind.image.load", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("image", image)
	defer span.End()

	shell.CheckInstalledSoftware("kind")

	loadArgs := []string{"load", "docker-image", image}
	// default cluster name is equals to 'kind'
	if c.kindName != "" {
		loadArgs = append(loadArgs, "--name", c.kindName)
	}

	result, err := shell.Execute(ctx, ".", "kind", loadArgs...)
	if err != nil {
		log.WithError(err).Fatal("Failed to load archive into kind")
	}
	log.WithFields(log.Fields{
		"image":  image,
		"result": result,
	}).Info("Image has been loaded into Kind runtime")

	return nil
}
