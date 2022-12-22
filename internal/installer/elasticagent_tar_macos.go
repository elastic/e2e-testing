// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/io"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/elastic/e2e-testing/pkg/downloads"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// elasticAgentTARDarwinPackage implements operations for a TAR installer
type elasticAgentTARDarwinPackage struct {
	elasticAgentPackage
}

// AttachElasticAgentTARDarwinPackage creates an instance for the TAR installer
func AttachElasticAgentTARDarwinPackage(d deploy.Deployment, service deploy.ServiceRequest) deploy.ServiceOperator {
	arch := "x86_64"
	if utils.GetArchitecture() == "arm64" {
		arch = "arm64"
	}

	return &elasticAgentTARDarwinPackage{
		elasticAgentPackage{
			service: service,
			deploy:  d,
			metadata: deploy.ServiceInstallerMetadata{
				AgentPath:     "/opt/Elastic/Agent",
				PackageType:   "tar",
				Os:            "darwin",
				Arch:          arch,
				FileExtension: "tar.gz",
				XPack:         true,
				Docker:        false,
			},
		},
	}
}

// AddFiles will add files into the service environment, default destination is /
func (i *elasticAgentTARDarwinPackage) AddFiles(ctx context.Context, files []string) error {
	return nil
}

// Inspect returns info on package
func (i *elasticAgentTARDarwinPackage) Inspect() (deploy.ServiceOperatorManifest, error) {
	return deploy.ServiceOperatorManifest{
		WorkDir:    i.metadata.AgentPath,
		CommitFile: "elastic-agent/.elastic-agent.active.commit",
	}, nil
}

// Install installs a package
func (i *elasticAgentTARDarwinPackage) Install(ctx context.Context) error {
	log.Trace("No TAR install instructions")
	return nil
}

// Exec will execute a command within the service environment
func (i *elasticAgentTARDarwinPackage) Exec(ctx context.Context, args []string) (string, error) {
	span, _ := apm.StartSpanOptions(ctx, "Executing Elastic Agent command", "elastic-agent.tar.exec", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", args)
	span.Context.SetLabel("runtime", runtime.GOOS)
	defer span.End()

	output, err := i.deploy.ExecIn(ctx, deploy.NewServiceRequest(common.FleetProfileName), i.service, args)
	return output, err
}

// Enroll will enroll the agent into fleet
func (i *elasticAgentTARDarwinPackage) Enroll(ctx context.Context, token string, extraFlags string) error {
	cmds := []string{"sudo", common.GetElasticAgentWorkingPath("elastic-agent", "elastic-agent"), "install"}
	span, _ := apm.StartSpanOptions(ctx, "Enrolling Elastic Agent with token", "elastic-agent.tar.enroll", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	span.Context.SetLabel("runtime", runtime.GOOS)
	defer span.End()

	cfg, _ := kibana.NewFleetConfig(token)
	cmds = append(cmds, cfg.Flags()...)
	if extraFlags != "" {
		cmds = append(cmds, extraFlags)
	}

	_, err := i.Exec(ctx, cmds)
	if err != nil {
		return fmt.Errorf("failed to install the agent with subcommand: %v", err)
	}
	return nil
}

// InstallCerts installs the certificates for a TAR package, using the right OS package manager
func (i *elasticAgentTARDarwinPackage) InstallCerts(ctx context.Context) error {
	return nil
}

// Logs prints logs of service
func (i *elasticAgentTARDarwinPackage) Logs(ctx context.Context) error {
	// TODO: we need to find a way to read MacOS logs for a service (the agent is installed under /Library/LaunchDaemons)
	// or we could read "/Library/Elastic/Agent/data/elastic-agent-*/logs/elastic-agent-json.log*"
	return i.deploy.Logs(ctx, i.service)
}

// Postinstall executes operations after installing a TAR package
func (i *elasticAgentTARDarwinPackage) Postinstall(ctx context.Context) error {
	return nil
}

// Preinstall executes operations before installing a TAR package
func (i *elasticAgentTARDarwinPackage) Preinstall(ctx context.Context) error {
	span, _ := apm.StartSpanOptions(ctx, "Pre-install operations for the Elastic Agent", "elastic-agent.tar.pre-install", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("runtime", runtime.GOOS)
	defer span.End()

	err := createAgentDirectories(ctx, i, []string{"sudo", "chown", "-R", "root:wheel", i.metadata.AgentPath})
	if err != nil {
		return err
	}

	// Idempotence: so no previous executions interfers with the current execution
	found, err := io.Exists(common.GetElasticAgentWorkingPath("elastic-agent"))
	if found && err == nil {
		err = os.RemoveAll(common.GetElasticAgentWorkingPath("elastic-agent"))
		if err != nil {
			log.Fatal("Could not remove elastic-agent.")
		}
		log.Trace("Cleared previously elastic-agent dir")
	}

	artifact := "elastic-agent"

	metadata := i.metadata

	_, binaryPath, err := downloads.FetchElasticArtifact(ctx, artifact, i.service.Version, metadata.Os, metadata.Arch, metadata.FileExtension, false, true)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":        artifact,
			"version":         i.service.Version,
			"packageMetadata": metadata,
			"error":           err,
		}).Error("Could not download the binary for the agent")
		return err
	}

	_, err = i.Exec(ctx, []string{"tar", "-zxf", binaryPath, "-C", common.GetElasticAgentWorkingPath()})
	if err != nil {
		return err
	}

	version := common.ElasticAgentVersion
	if downloads.IsAlias(version) {
		v, err := downloads.GetElasticArtifactVersion(version)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"version": version,
			}).Warn("Failed to get the version, keeping current version")
		} else {
			version = v
		}
	}

	srcPath := common.GetElasticAgentWorkingPath(fmt.Sprintf("%s-%s-%s-%s", artifact, downloads.GetSnapshotVersion(version), metadata.Os, metadata.Arch))
	_, _ = i.Exec(ctx, []string{"rm", "-fr", common.GetElasticAgentWorkingPath("elastic-agent")})
	output, _ := i.Exec(ctx, []string{"mv", "-f", srcPath, common.GetElasticAgentWorkingPath("elastic-agent")})
	log.WithField("output", output).Trace("Moved elastic-agent")
	return nil
}

// Restart will restart a service
func (i *elasticAgentTARDarwinPackage) Restart(ctx context.Context) error {
	err := i.Stop(ctx)
	if err != nil {
		return err
	}
	return i.Start(ctx)
}

// Start will start a service
func (i *elasticAgentTARDarwinPackage) Start(ctx context.Context) error {
	cmds := []string{"launchctl", "start", "elastic-agent"}
	span, _ := apm.StartSpanOptions(ctx, "Starting Elastic Agent service", "elastic-agent.tar.start", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	span.Context.SetLabel("runtime", runtime.GOOS)
	defer span.End()

	_, err := i.Exec(ctx, cmds)
	if err != nil {
		return err
	}
	return nil
}

// Stop will start a service
func (i *elasticAgentTARDarwinPackage) Stop(ctx context.Context) error {
	cmds := []string{"launchctl", "stop", "elastic-agent"}
	span, _ := apm.StartSpanOptions(ctx, "Stopping Elastic Agent service", "elastic-agent.tar.stop", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	span.Context.SetLabel("runtime", runtime.GOOS)
	defer span.End()

	_, err := i.Exec(ctx, cmds)
	if err != nil {
		return err
	}
	return nil
}

// Uninstall uninstalls a TAR package
func (i *elasticAgentTARDarwinPackage) Uninstall(ctx context.Context) error {
	cmds := []string{"sudo", common.GetElasticAgentWorkingPath("elastic-agent", "elastic-agent"), "uninstall", "-f"}
	span, _ := apm.StartSpanOptions(ctx, "Uninstalling Elastic Agent", "elastic-agent.tar.uninstall", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	span.Context.SetLabel("runtime", runtime.GOOS)
	defer span.End()
	_, err := i.Exec(ctx, cmds)
	if err != nil {
		return fmt.Errorf("failed to uninstall the agent with subcommand: %v", err)
	}
	return nil
}

// Upgrade upgrades a TAR package
func (i *elasticAgentTARDarwinPackage) Upgrade(ctx context.Context, version string) error {
	return doUpgrade(ctx, i)
}
