// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"context"
	"fmt"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// elasticAgentTARPackage implements operations for a RPM installer
type elasticAgentTARPackage struct {
	service deploy.ServiceRequest
	deploy  deploy.Deployment
}

// AttachElasticAgentTARPackage creates an instance for the RPM installer
func AttachElasticAgentTARPackage(deploy deploy.Deployment, service deploy.ServiceRequest) deploy.ServiceOperator {
	return &elasticAgentTARPackage{
		service: service,
		deploy:  deploy,
	}
}

// AddFiles will add files into the service environment, default destination is /
func (i *elasticAgentTARPackage) AddFiles(ctx context.Context, files []string) error {
	span, _ := apm.StartSpanOptions(ctx, "Adding files to the Elastic Agent", "elastic-agent.tar.add-files", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("files", files)
	defer span.End()

	return i.deploy.AddFiles(ctx, common.FleetProfileServiceRequest, i.service, files)
}

// Inspect returns info on package
func (i *elasticAgentTARPackage) Inspect() (deploy.ServiceOperatorManifest, error) {
	return deploy.ServiceOperatorManifest{
		WorkDir:    "/opt/Elastic/Agent",
		CommitFile: "/elastic-agent/.elastic-agent.active.commit",
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

	output, err := i.deploy.ExecIn(ctx, common.FleetProfileServiceRequest, i.service, args)
	return output, err
}

// Enroll will enroll the agent into fleet
func (i *elasticAgentTARPackage) Enroll(ctx context.Context, token string) error {
	cmds := []string{"/elastic-agent/elastic-agent", "install"}
	span, _ := apm.StartSpanOptions(ctx, "Enrolling Elastic Agent with token", "elastic-agent.tar.enroll", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	defer span.End()

	cfg, _ := kibana.NewFleetConfig(token)
	for _, arg := range cfg.Flags() {
		cmds = append(cmds, arg)
	}

	_, err := i.Exec(ctx, cmds)
	if err != nil {
		return fmt.Errorf("Failed to install the agent with subcommand: %v", err)
	}
	return nil
}

// InstallCerts installs the certificates for a TAR package, using the right OS package manager
func (i *elasticAgentTARPackage) InstallCerts(ctx context.Context) error {
	return nil
}

// Logs prints logs of service
func (i *elasticAgentTARPackage) Logs() error {
	return i.deploy.Logs(i.service)
}

// Postinstall executes operations after installing a TAR package
func (i *elasticAgentTARPackage) Postinstall(ctx context.Context) error {
	return nil
}

// Preinstall executes operations before installing a TAR package
func (i *elasticAgentTARPackage) Preinstall(ctx context.Context) error {
	span, _ := apm.StartSpanOptions(ctx, "Pre-install operations for the Elastic Agent", "elastic-agent.tar.pre-install", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	artifact := "elastic-agent"
	os := "linux"
	arch := "x86_64"
	if utils.GetArchitecture() == "arm64" {
		arch = "arm64"
	}
	extension := "tar.gz"

	_, binaryPath, err := utils.FetchElasticArtifact(ctx, artifact, common.BeatVersion, os, arch, extension, false, true)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":  artifact,
			"version":   common.BeatVersion,
			"os":        os,
			"arch":      arch,
			"extension": extension,
			"error":     err,
		}).Error("Could not download the binary for the agent")
		return err
	}

	err = i.AddFiles(context.Background(), []string{binaryPath})
	if err != nil {
		return err
	}

	output, _ := i.Exec(ctx, []string{"mv", fmt.Sprintf("/%s-%s-%s-%s", artifact, common.BeatVersion, os, arch), "/elastic-agent"})
	log.WithField("output", output).Trace("Moved elastic-agent")
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
	cmds := []string{"elastic-agent", "uninstall", "-f"}
	span, _ := apm.StartSpanOptions(ctx, "Uninstalling Elastic Agent", "elastic-agent.tar.uninstall", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	defer span.End()
	_, err := i.Exec(ctx, cmds)
	if err != nil {
		return fmt.Errorf("Failed to uninstall the agent with subcommand: %v", err)
	}
	return nil
}
