// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"context"
	"fmt"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/elastic/e2e-testing/pkg/downloads"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm/v2"
)

// elasticAgentDockerPackage implements operations for a docker installer
type elasticAgentDockerPackage struct {
	elasticAgentPackage
}

// AttachElasticAgentDockerPackage creates an instance for the docker installer
func AttachElasticAgentDockerPackage(d deploy.Deployment, service deploy.ServiceRequest) deploy.ServiceOperator {
	return &elasticAgentDockerPackage{
		elasticAgentPackage{
			service: service,
			deploy:  d,
			metadata: deploy.ServiceInstallerMetadata{
				AgentPath:     "/usr/share/elastic-agent",
				PackageType:   "docker",
				Os:            "linux",
				Arch:          utils.GetArchitecture(),
				FileExtension: "tar.gz",
				XPack:         true,
				Docker:        true,
			},
		},
	}
}

// AddFiles will add files into the service environment, default destination is /
func (i *elasticAgentDockerPackage) AddFiles(ctx context.Context, files []string) error {
	span, _ := apm.StartSpanOptions(ctx, "Adding files to the Elastic Agent", "elastic-agent.docker.add-files", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("files", files)
	defer span.End()

	return i.deploy.AddFiles(ctx, deploy.NewServiceRequest(common.FleetProfileName), i.service, files)
}

// Inspect returns info on package
func (i *elasticAgentDockerPackage) Inspect() (deploy.ServiceOperatorManifest, error) {
	return deploy.ServiceOperatorManifest{
		WorkDir:    i.metadata.AgentPath,
		CommitFile: "/usr/share/elastic-agent/.elastic-agent.active.commit",
	}, nil
}

// Install installs a package
func (i *elasticAgentDockerPackage) Install(ctx context.Context) error {
	return nil
}

// Exec will execute a command within the service environment
func (i *elasticAgentDockerPackage) Exec(ctx context.Context, args []string) (string, error) {
	span, _ := apm.StartSpanOptions(ctx, "Executing Elastic Agent command", "elastic-agent.docker.exec", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", args)
	defer span.End()

	output, err := i.deploy.ExecIn(ctx, deploy.NewServiceRequest(common.FleetProfileName), i.service, args)
	return output, err
}

// Enroll will enroll the agent into fleet
func (i *elasticAgentDockerPackage) Enroll(ctx context.Context, token string, extraFlags string) error {
	return nil
}

// InstallCerts installs the certificates for a package, using the right OS package manager
func (i *elasticAgentDockerPackage) InstallCerts(ctx context.Context) error {
	return nil
}

// Logs prints logs of service
func (i *elasticAgentDockerPackage) Logs(ctx context.Context) error {
	return i.deploy.Logs(ctx, i.service)
}

// Postinstall executes operations after installing a package
func (i *elasticAgentDockerPackage) Postinstall(ctx context.Context) error {
	return nil
}

// Preinstall executes operations before installing a package
func (i *elasticAgentDockerPackage) Preinstall(ctx context.Context) error {
	span, _ := apm.StartSpanOptions(ctx, "Pre-install operations for the Elastic Agent", "elastic-agent.docker.pre-install", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	// handle ubi8 images
	artifact := "elastic-agent" + common.ProfileEnv["elasticAgentDockerImageSuffix"]

	metadata := i.metadata

	_, binaryPath, err := downloads.FetchElasticArtifact(ctx, artifact, i.service.Version, metadata.Os, metadata.Arch, metadata.FileExtension, metadata.Docker, metadata.XPack)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":        artifact,
			"version":         i.service.Version,
			"packageMetadata": metadata,
			"error":           err,
		}).Error("Could not download the binary for the agent")
		return err
	}

	err = deploy.LoadImage(binaryPath)
	if err != nil {
		return err
	}

	// we need to tag the loaded image because its tag relates to the target branch
	return deploy.TagImage(
		fmt.Sprintf("docker.elastic.co/beats/%s:%s", artifact, downloads.GetSnapshotVersion(common.BeatVersionBase)),
		fmt.Sprintf("docker.elastic.co/observability-ci/%s:%s-%s", artifact, downloads.GetSnapshotVersion(common.ElasticAgentVersion), metadata.Arch),
		// tagging including git commit and snapshot
		fmt.Sprintf("docker.elastic.co/observability-ci/%s:%s-%s", artifact, downloads.GetFullVersion(common.ElasticAgentVersion), metadata.Arch),
	)
}

// Restart will restart a service
func (i *elasticAgentDockerPackage) Restart(ctx context.Context) error {
	cmds := []string{"systemctl", "restart", "elastic-agent"}
	span, _ := apm.StartSpanOptions(ctx, "Restarting Elastic Agent service", "elastic-agent.docker.restart", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	defer span.End()

	_, err := i.Exec(ctx, cmds)
	if err != nil {
		return err
	}
	return nil
}

// Start will start a service
func (i *elasticAgentDockerPackage) Start(ctx context.Context) error {
	cmds := []string{"systemctl", "start", "elastic-agent"}
	span, _ := apm.StartSpanOptions(ctx, "Starting Elastic Agent service", "elastic-agent.docker.start", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	defer span.End()

	_, err := i.Exec(ctx, cmds)
	if err != nil {
		return err
	}
	return nil
}

// Stop will start a service
func (i *elasticAgentDockerPackage) Stop(ctx context.Context) error {
	cmds := []string{"systemctl", "stop", "elastic-agent"}
	span, _ := apm.StartSpanOptions(ctx, "Stopping Elastic Agent service", "elastic-agent.docker.stop", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	defer span.End()

	_, err := i.Exec(ctx, cmds)
	if err != nil {
		return err
	}
	return nil
}

// Uninstall uninstalls a Docker package
func (i *elasticAgentDockerPackage) Uninstall(ctx context.Context) error {
	cmds := []string{"elastic-agent", "uninstall", "-f"}
	span, _ := apm.StartSpanOptions(ctx, "Uninstalling Elastic Agent", "elastic-agent.docker.uninstall", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	defer span.End()

	_, err := i.Exec(ctx, cmds)
	if err != nil {
		return fmt.Errorf("failed to uninstall the agent with subcommand: %v", err)
	}
	return nil
}

// Upgrade upgrades a Docker package
func (i *elasticAgentDockerPackage) Upgrade(ctx context.Context, version string) error {
	return doUpgrade(ctx, i)
}
