package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/elastic/e2e-testing/cli/shell"
)

type kubernetesControl struct {
	config    string
	Namespace string
}

func (c kubernetesControl) WithConfig(config string) kubernetesControl {
	c.config = config
	return c
}

func (c kubernetesControl) WithNamespace(ctx context.Context, namespace string) kubernetesControl {
	if namespace == "" {
		namespace = "test-" + uuid.New().String()
		output, err := c.Run(ctx, "create", "namespace", namespace)
		if err != nil {
			log.WithError(err).WithField("output", output).Warn(
				"failed to create namespace, default will be used")
		}
	}
	c.Namespace = namespace
	return c
}

func (c kubernetesControl) Cleanup(ctx context.Context) error {
	if c.Namespace != "" {
		output, err := c.Run(ctx, "delete", "namespace", c.Namespace)
		if err != nil {
			return fmt.Errorf("failed to delete namespace %s: %v: %s", c.Namespace, err, output)
		}
	}
	return nil
}

func (c kubernetesControl) Run(ctx context.Context, args ...string) (output string, err error) {
	shell.CheckInstalledSoftware("kubectl")
	if c.config != "" {
		args = append(args, "--kubeconfig", c.config)
	}
	if c.Namespace != "" {
		args = append(args, "--namespace", c.Namespace)
	}
	return shell.Execute(ctx, ".", "kubectl", args...)
}

type kubernetesCluster struct {
	kindName   string
	kubeconfig string

	tmpDir string
}

func (c kubernetesCluster) Kubectl() kubernetesControl {
	return kubernetesControl{}.WithConfig(c.kubeconfig)
}

func (c kubernetesCluster) isAvailable(ctx context.Context) error {
	_, err := c.Kubectl().Run(ctx, "api-versions")
	return err
}

func (c *kubernetesCluster) initialize(ctx context.Context) error {
	err := c.isAvailable(ctx)
	if err == nil {
		return nil
	}

	log.Info("Kubernetes cluster not available, will start one using kind")
	shell.CheckInstalledSoftware("kind")

	c.tmpDir, err = ioutil.TempDir(os.TempDir(), "test-")
	if err != nil {
		log.WithError(err).Fatal("Failed to create temporary directory")
	}

	name := "kind-" + uuid.New().String()
	c.kubeconfig = filepath.Join(c.tmpDir, "kubeconfig")

	output, err := shell.Execute(ctx, ".", "kind", "create", "cluster",
		"--name", name,
		"--kubeconfig", c.kubeconfig,
	)
	if err != nil {
		log.WithError(err).WithField("output", output).Fatal("Failed to create kind cluster")
		return err
	}
	c.kindName = name

	return nil
}

func (c *kubernetesCluster) cleanup(ctx context.Context) {
	if c.kindName != "" {
		_, err := shell.Execute(ctx, ".", "kind", "delete", "cluster", "--name", c.kindName)
		if err != nil {
			log.Warnf("Failed to delete kind cluster %s", c.kindName)
		}
		c.kindName = ""
	}
	if c.tmpDir != "" {
		err := os.RemoveAll(c.tmpDir)
		if err != nil {
			log.Warnf("Failed to remove temporary directory %s", c.tmpDir)
		}
	}
}
