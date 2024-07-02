// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	"github.com/elastic/e2e-testing/pkg/downloads"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm/v2"
)

// elasticAgentRPMPackage implements operations for a RPM installer
type elasticAgentRPMPackage struct {
	elasticAgentPackage
}

// AttachElasticAgentRPMPackage creates an instance for the RPM installer
func AttachElasticAgentRPMPackage(d deploy.Deployment, service deploy.ServiceRequest) deploy.ServiceOperator {
	arch := "x86_64"
	if utils.GetArchitecture() == "arm64" {
		arch = "aarch64"
	}

	return &elasticAgentRPMPackage{
		elasticAgentPackage{
			service: service,
			deploy:  d,
			metadata: deploy.ServiceInstallerMetadata{
				AgentPath:     "/var/lib/elastic-agent",
				PackageType:   "rpm",
				Os:            "linux",
				Arch:          arch,
				FileExtension: "rpm",
				XPack:         true,
				Docker:        false,
			},
		},
	}
}

// AddFiles will add files into the service environment, default destination is /
func (i *elasticAgentRPMPackage) AddFiles(ctx context.Context, files []string) error {
	span, _ := apm.StartSpanOptions(ctx, "Adding files to the Elastic Agent", "elastic-agent.rpm.add-files", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("files", files)
	defer span.End()

	return i.deploy.AddFiles(ctx, deploy.NewServiceRequest(common.FleetProfileName), i.service, files)
}

// Inspect returns info on package
func (i *elasticAgentRPMPackage) Inspect() (deploy.ServiceOperatorManifest, error) {
	return deploy.ServiceOperatorManifest{
		WorkDir:    i.metadata.AgentPath,
		CommitFile: "/etc/elastic-agent/.elastic-agent.active.commit",
	}, nil
}

// Install installs a RPM package
func (i *elasticAgentRPMPackage) Install(ctx context.Context) error {
	log.Trace("No additional install commands for RPM")
	return nil
}

// Exec will execute a command within the service environment
func (i *elasticAgentRPMPackage) Exec(ctx context.Context, args []string) (string, error) {
	span, _ := apm.StartSpanOptions(ctx, "Executing Elastic Agent command", "elastic-agent.rpm.exec", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", args)
	defer span.End()

	output, err := i.deploy.ExecIn(ctx, deploy.NewServiceRequest(common.FleetProfileName), i.service, args)
	return output, err
}

// Enroll will enroll the agent into fleet
func (i *elasticAgentRPMPackage) Enroll(ctx context.Context, token string, extraFlags string) error {
	cmds := []string{"elastic-agent", "enroll"}
	span, _ := apm.StartSpanOptions(ctx, "Enrolling Elastic Agent with token", "elastic-agent.rpm.enroll", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	defer span.End()

	cfg, _ := kibana.NewFleetConfig(token)
	cmds = append(cmds, cfg.Flags()...)
	if extraFlags != "" {
		cmds = append(cmds, extraFlags)
	}

	output, err := i.Exec(ctx, cmds)
	log.Trace(output)
	if err != nil {
		return fmt.Errorf("failed to install the agent with subcommand: %v", err)
	}
	return nil
}

// InstallCerts installs the certificates for a RPM package, using the right OS package manager
func (i *elasticAgentRPMPackage) InstallCerts(ctx context.Context) error {
	span, _ := apm.StartSpanOptions(ctx, "Installing certificates for the Elastic Agent", "elastic-agent.rpm.install-certs", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	cmds := [][]string{
		{"yum", "check-update"},
		{"yum", "install", "ca-certificates", "-y"},
		{"update-ca-trust", "force-enable"},
		{"update-ca-trust", "extract"},
	}
	for _, cmd := range cmds {
		if _, err := i.Exec(ctx, cmd); err != nil {
			return err
		}
	}
	return nil
}

// Logs prints logs of service
func (i *elasticAgentRPMPackage) Logs(ctx context.Context) error {
	// TODO we could read "/var/lib/elastic-agent/data/elastic-agent-*/logs/elastic-agent-json.log"
	return systemCtlLog(ctx, "rpm", i.Exec)
}

// Postinstall executes operations after installing a RPM package
func (i *elasticAgentRPMPackage) Postinstall(ctx context.Context) error {
	for _, bp := range i.service.BackgroundProcesses {
		if strings.EqualFold(bp, "filebeat") || strings.EqualFold(bp, "metricbeat") {
			// post-install the dependant binary first
			err := systemCtlPostInstall(ctx, "centos", bp, i.Exec)
			if err != nil {
				return err
			}
		}
	}

	return systemCtlPostInstall(ctx, "centos", "elastic-agent", i.Exec)
}

// Preinstall executes operations before installing a RPM package
func (i *elasticAgentRPMPackage) Preinstall(ctx context.Context) error {
	err := createAgentDirectories(ctx, i, []string{"sudo", "chown", "-R", "root:root", i.metadata.AgentPath})
	if err != nil {
		return err
	}

	installArtifactFn := func(ctx context.Context, artifact string, version string, useCISnapshots bool) error {
		span, _ := apm.StartSpanOptions(ctx, "Pre-install "+artifact, artifact+".rpm.pre-install", apm.SpanOptions{
			Parent: apm.SpanFromContext(ctx).TraceContext(),
		})
		defer span.End()

		metadata := i.metadata

		binaryName, binaryPath, err := downloads.FetchElasticArtifactForSnapshots(ctx, useCISnapshots, artifact, version, metadata.Os, metadata.Arch, metadata.FileExtension, metadata.Docker, metadata.XPack)
		if err != nil {
			log.WithFields(log.Fields{
				"artifact":        artifact,
				"version":         version,
				"packageMetadata": metadata,
				"error":           err,
			}).Error("Could not download the binary")
			return err
		}

		err = i.AddFiles(ctx, []string{binaryPath})
		if err != nil {
			return err
		}

		_, err = i.Exec(ctx, []string{"yum", "localinstall", "/" + binaryName, "-y"})
		if err != nil {
			return err
		}

		return nil
	}

	for _, bp := range i.service.BackgroundProcesses {
		if strings.EqualFold(bp, "filebeat") || strings.EqualFold(bp, "metricbeat") {
			// pre-install the dependant binary first, using the stack version
			err := installArtifactFn(ctx, bp, common.BeatVersion, downloads.UseBeatsCISnapshots())
			if err != nil {
				return err
			}
		}
	}

	return installArtifactFn(ctx, "elastic-agent", i.service.Version, downloads.UseElasticAgentCISnapshots())
}

// Restart will restart a service
func (i *elasticAgentRPMPackage) Restart(ctx context.Context) error {
	for _, bp := range i.service.BackgroundProcesses {
		if strings.EqualFold(bp, "filebeat") || strings.EqualFold(bp, "metricbeat") {
			// start the dependant binary first
			err := systemCtlRestart(ctx, "centos", bp, i.Exec)
			if err != nil {
				return err
			}
		}
	}

	return systemCtlRestart(ctx, "centos", "elastic-agent", i.Exec)
}

// Start will start a service
func (i *elasticAgentRPMPackage) Start(ctx context.Context) error {
	for _, bp := range i.service.BackgroundProcesses {
		if strings.EqualFold(bp, "filebeat") || strings.EqualFold(bp, "metricbeat") {
			// start the dependant binary first
			err := systemCtlStart(ctx, "centos", bp, i.Exec)
			if err != nil {
				return err
			}
		}
	}

	return systemCtlStart(ctx, "centos", "elastic-agent", i.Exec)
}

// Stop will start a service
func (i *elasticAgentRPMPackage) Stop(ctx context.Context) error {
	cmds := []string{"systemctl", "stop", "elastic-agent"}
	span, _ := apm.StartSpanOptions(ctx, "Stopping Elastic Agent service", "elastic-agent.rpm.stop", apm.SpanOptions{
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

// Uninstall uninstalls a RPM package
func (i *elasticAgentRPMPackage) Uninstall(ctx context.Context) error {
	cmds := []string{"elastic-agent", "uninstall", "-f"}
	span, _ := apm.StartSpanOptions(ctx, "Uninstalling Elastic Agent", "elastic-agent.rpm.uninstall", apm.SpanOptions{
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

// Upgrade upgrades a RPM package
func (i *elasticAgentRPMPackage) Upgrade(ctx context.Context, version string) error {
	return doUpgrade(ctx, i)
}
