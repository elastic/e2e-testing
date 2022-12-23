// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/io"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/elastic/e2e-testing/pkg/downloads"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// elasticAgentTARPackage implements operations for a RPM installer
type elasticAgentTARPackage struct {
	elasticAgentPackage
}

// AttachElasticAgentTARPackage creates an instance for the RPM installer
func AttachElasticAgentTARPackage(d deploy.Deployment, service deploy.ServiceRequest) deploy.ServiceOperator {
	arch := "x86_64"
	if utils.GetArchitecture() == "arm64" {
		arch = "arm64"
	}

	return &elasticAgentTARPackage{
		elasticAgentPackage{
			service: service,
			deploy:  d,
			metadata: deploy.ServiceInstallerMetadata{
				AgentPath:     "/opt/Elastic/Agent",
				PackageType:   "tar",
				Os:            "linux",
				Arch:          arch,
				FileExtension: "tar.gz",
				XPack:         true,
				Docker:        false,
			},
		},
	}
}

// AddFiles will add files into the service environment, default destination is /
func (i *elasticAgentTARPackage) AddFiles(ctx context.Context, files []string) error {
	span, _ := apm.StartSpanOptions(ctx, "Adding files to the Elastic Agent", "elastic-agent.tar.add-files", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("files", files)
	defer span.End()

	return i.deploy.AddFiles(ctx, deploy.NewServiceRequest(common.FleetProfileName), i.service, files)
}

// Inspect returns info on package
func (i *elasticAgentTARPackage) Inspect() (deploy.ServiceOperatorManifest, error) {
	return deploy.ServiceOperatorManifest{
		WorkDir:    i.metadata.AgentPath,
		CommitFile: "elastic-agent/.elastic-agent.active.commit",
	}, nil
}

// Install installs a TAR package
func (i *elasticAgentTARPackage) Install(ctx context.Context) error {
	log.Trace("No TAR install instructions")
	return nil
}

// Exec will execute a command within the service environment
func (i *elasticAgentTARPackage) Exec(ctx context.Context, args []string) (string, error) {
	span, _ := apm.StartSpanOptions(ctx, "Executing Elastic Agent command", "elastic-agent.tar.exec", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", args)
	defer span.End()

	output, err := i.deploy.ExecIn(ctx, deploy.NewServiceRequest(common.FleetProfileName), i.service, args)
	return output, err
}

// Enroll will enroll the agent into fleet
func (i *elasticAgentTARPackage) Enroll(ctx context.Context, token string, extraFlags string) error {
	cmds := []string{common.GetElasticAgentWorkingPath("elastic-agent", "elastic-agent"), "install"}
	span, _ := apm.StartSpanOptions(ctx, "Enrolling Elastic Agent with token", "elastic-agent.tar.enroll", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
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
func (i *elasticAgentTARPackage) InstallCerts(ctx context.Context) error {
	return nil
}

// Logs prints logs of service
func (i *elasticAgentTARPackage) Logs(ctx context.Context) error {
	// TODO: we could read "/opt/Elastic/Agent/data/elastic-agent-*/logs/elastic-agent-json.log*"
	return systemCtlLog(ctx, "tar", i.Exec)
}

// Postinstall executes operations after installing a TAR package
func (i *elasticAgentTARPackage) Postinstall(ctx context.Context) error {
	return nil
}

// Preinstall executes operations before installing a TAR package
func (i *elasticAgentTARPackage) Preinstall(ctx context.Context) error {
	err := createAgentDirectories(ctx, i, []string{"sudo", "chown", "-R", "root:root", i.metadata.AgentPath})
	if err != nil {
		return err
	}

	installArtifactFn := func(ctx context.Context, artifact string, version string, useCISnapshots bool) error {
		span, _ := apm.StartSpanOptions(ctx, "Pre-install "+artifact, artifact+".tar.pre-install", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		defer span.End()

		found, err := io.Exists(artifact)
		if found && err == nil {
			err = os.RemoveAll(artifact)
			if err != nil {
				log.Fatal("Could not remove artifact directory for reinitialization.")
			}
			log.Trace("Cleared previously downloaded artifacts")
		}

		metadata := i.metadata

		_, binaryPath, err := downloads.FetchElasticArtifactForSnapshots(ctx, useCISnapshots, artifact, version, metadata.Os, metadata.Arch, metadata.FileExtension, false, true)
		if err != nil {
			log.WithFields(log.Fields{
				"artifact":        artifact,
				"version":         version,
				"packageMetadata": metadata,
				"error":           err,
			}).Error("Could not download the binary")
			return err
		}

		_, err = i.Exec(ctx, []string{"tar", "-zxf", binaryPath, "-C", common.GetElasticAgentWorkingPath()})
		if err != nil {
			return err
		}

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
		_, _ = i.Exec(ctx, []string{"rm", "-fr", common.GetElasticAgentWorkingPath(artifact)})
		output, _ := i.Exec(ctx, []string{"mv", "-f", srcPath, common.GetElasticAgentWorkingPath(artifact)})
		log.WithFields(log.Fields{
			"output":   output,
			"artifact": artifact,
		}).Trace("Moved")
		return nil
	}

	for _, bp := range i.service.BackgroundProcesses {
		if strings.EqualFold(bp, "filebeat") || strings.EqualFold(bp, "metricbeat") {
			// pre-install the dependant binary first
			err := installArtifactFn(ctx, bp, common.BeatVersion, downloads.UseBeatsCISnapshots())
			if err != nil {
				return err
			}
		}
	}

	return installArtifactFn(ctx, "elastic-agent", i.service.Version, downloads.UseElasticAgentCISnapshots())

}

// Restart will restart a service
func (i *elasticAgentTARPackage) Restart(ctx context.Context) error {
	cmds := []string{"systemctl", "restart", "elastic-agent"}
	span, _ := apm.StartSpanOptions(ctx, "Restarting Elastic Agent service", "elastic-agent.tar.restart", apm.SpanOptions{
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
func (i *elasticAgentTARPackage) Start(ctx context.Context) error {
	cmds := []string{"systemctl", "start", "elastic-agent"}
	span, _ := apm.StartSpanOptions(ctx, "Starting Elastic Agent service", "elastic-agent.tar.start", apm.SpanOptions{
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
func (i *elasticAgentTARPackage) Stop(ctx context.Context) error {
	cmds := []string{"systemctl", "stop", "elastic-agent"}
	span, _ := apm.StartSpanOptions(ctx, "Stopping Elastic Agent service", "elastic-agent.tar.stop", apm.SpanOptions{
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

// Uninstall uninstalls a TAR package
func (i *elasticAgentTARPackage) Uninstall(ctx context.Context) error {
	cmds := []string{"/opt/Elastic/Agent/elastic-agent", "uninstall", "-f"}
	span, _ := apm.StartSpanOptions(ctx, "Uninstalling Elastic Agent", "elastic-agent.tar.uninstall", apm.SpanOptions{
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

// Upgrade upgrades a TAR package
func (i *elasticAgentTARPackage) Upgrade(ctx context.Context, version string) error {
	return doUpgrade(ctx, i)
}
