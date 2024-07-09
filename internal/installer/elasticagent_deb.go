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

// elasticAgentDEBPackage implements operations for a DEB installer
type elasticAgentDEBPackage struct {
	elasticAgentPackage
}

// AttachElasticAgentDEBPackage creates an instance for the DEB installer
func AttachElasticAgentDEBPackage(d deploy.Deployment, service deploy.ServiceRequest) deploy.ServiceOperator {
	return &elasticAgentDEBPackage{
		elasticAgentPackage{
			service: service,
			deploy:  d,
			metadata: deploy.ServiceInstallerMetadata{
				AgentPath:     "/var/lib/elastic-agent",
				PackageType:   "deb",
				Os:            "linux",
				Arch:          utils.GetArchitecture(),
				FileExtension: "deb",
				XPack:         true,
				Docker:        false,
			},
		},
	}
}

// AddFiles will add files into the service environment, default destination is /
func (i *elasticAgentDEBPackage) AddFiles(ctx context.Context, files []string) error {
	span, _ := apm.StartSpanOptions(ctx, "Adding files to the Elastic Agent", "elastic-agent.debian.add-files", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("files", files)
	defer span.End()

	return i.deploy.AddFiles(ctx, deploy.NewServiceRequest(common.FleetProfileName), i.service, files)
}

// Inspect returns info on package
func (i *elasticAgentDEBPackage) Inspect() (deploy.ServiceOperatorManifest, error) {
	return deploy.ServiceOperatorManifest{
		WorkDir:    i.metadata.AgentPath,
		CommitFile: "/etc/elastic-agent/.elastic-agent.active.commit",
	}, nil
}

// Install installs a DEB package
func (i *elasticAgentDEBPackage) Install(ctx context.Context) error {
	log.Trace("No additional install commands for DEB")
	return nil
}

// Exec will execute a command within the service environment
func (i *elasticAgentDEBPackage) Exec(ctx context.Context, args []string) (string, error) {
	span, _ := apm.StartSpanOptions(ctx, "Executing Elastic Agent command", "elastic-agent.debian.exec", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", args)
	defer span.End()

	output, err := i.deploy.ExecIn(ctx, deploy.NewServiceRequest(common.FleetProfileName), i.service, args)
	return output, err
}

// Enroll will enroll the agent into fleet
func (i *elasticAgentDEBPackage) Enroll(ctx context.Context, token string, extraFlags string) error {
	cmds := []string{"elastic-agent", "enroll"}
	span, _ := apm.StartSpanOptions(ctx, "Enrolling Elastic Agent with token", "elastic-agent.debian.enroll", apm.SpanOptions{
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

// InstallCerts installs the certificates for a DEB package, using the right OS package manager
func (i *elasticAgentDEBPackage) InstallCerts(ctx context.Context) error {
	span, _ := apm.StartSpanOptions(ctx, "Installing certificates for the Elastic Agent", "elastic-agent.debian.install-certs", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	cmds := [][]string{
		{"apt-get", "update"},
		{"apt", "install", "ca-certificates", "-y"},
		{"update-ca-certificates", "-f"},
	}
	for _, cmd := range cmds {
		if _, err := i.Exec(ctx, cmd); err != nil {
			return err
		}
	}
	return nil
}

// Logs prints logs of service
func (i *elasticAgentDEBPackage) Logs(ctx context.Context) error {
	// TODO we could read "/var/lib/elastic-agent/data/elastic-agent-*/logs/elastic-agent-json.log"
	return systemCtlLog(ctx, "debian", i.Exec)
}

// Postinstall executes operations after installing a DEB package
func (i *elasticAgentDEBPackage) Postinstall(ctx context.Context) error {
	for _, bp := range i.service.BackgroundProcesses {
		if strings.EqualFold(bp, "filebeat") || strings.EqualFold(bp, "metricbeat") {
			// post-install the dependant binary first
			err := systemCtlPostInstall(ctx, "debian", bp, i.Exec)
			if err != nil {
				return err
			}
		}
	}

	return systemCtlPostInstall(ctx, "debian", "elastic-agent", i.Exec)
}

// Preinstall executes operations before installing a DEB package
func (i *elasticAgentDEBPackage) Preinstall(ctx context.Context) error {
	err := createAgentDirectories(ctx, i, []string{"sudo", "chown", "-R", "root:root", i.metadata.AgentPath})
	if err != nil {
		return err
	}

	installArtifactFn := func(ctx context.Context, artifact string, version string, useCISnapshots bool) error {
		span, _ := apm.StartSpanOptions(ctx, "Pre-install "+artifact, artifact+".debian.pre-install", apm.SpanOptions{
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
			}).Error("Could not download the binary for the agent")
			return err
		}

		err = i.AddFiles(ctx, []string{binaryPath})
		if err != nil {
			return err
		}

		_, err = i.Exec(ctx, []string{"apt", "install", "/" + binaryName, "-y"})
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
func (i *elasticAgentDEBPackage) Restart(ctx context.Context) error {
	for _, bp := range i.service.BackgroundProcesses {
		if strings.EqualFold(bp, "filebeat") || strings.EqualFold(bp, "metricbeat") {
			// restart the dependant binary first
			err := systemCtlRestart(ctx, "debian", bp, i.Exec)
			if err != nil {
				return err
			}
		}
	}

	return systemCtlRestart(ctx, "debian", "elastic-agent", i.Exec)
}

// Start will start a service
func (i *elasticAgentDEBPackage) Start(ctx context.Context) error {
	for _, bp := range i.service.BackgroundProcesses {
		if strings.EqualFold(bp, "filebeat") || strings.EqualFold(bp, "metricbeat") {
			// start the dependant binary first
			err := systemCtlStart(ctx, "debian", bp, i.Exec)
			if err != nil {
				return err
			}
		}
	}

	return systemCtlStart(ctx, "debian", "elastic-agent", i.Exec)
}

// Stop will start a service
func (i *elasticAgentDEBPackage) Stop(ctx context.Context) error {
	cmds := []string{"systemctl", "stop", "elastic-agent"}
	span, _ := apm.StartSpanOptions(ctx, "Stopping Elastic Agent service", "elastic-agent.debian.stop", apm.SpanOptions{
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

// Uninstall uninstalls a DEB package
func (i *elasticAgentDEBPackage) Uninstall(ctx context.Context) error {
	cmds := []string{"elastic-agent", "uninstall", "-f"}
	span, _ := apm.StartSpanOptions(ctx, "Uninstalling Elastic Agent", "elastic-agent.debian.uninstall", apm.SpanOptions{
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

// Upgrade upgrade a DEB package
func (i *elasticAgentDEBPackage) Upgrade(ctx context.Context, version string) error {
	return doUpgrade(ctx, i)
}
