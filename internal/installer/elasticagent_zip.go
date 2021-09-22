// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"context"
	"fmt"
	"strings"

	elasticversion "github.com/elastic/e2e-testing/internal"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/kibana"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// elasticAgentZIPPackage implements operations for a ZIP installer
type elasticAgentZIPPackage struct {
	service deploy.ServiceRequest
	deploy  deploy.Deployment
}

// AttachElasticAgentZIPPackage creates an instance for the ZIP installer
func AttachElasticAgentZIPPackage(deploy deploy.Deployment, service deploy.ServiceRequest) deploy.ServiceOperator {
	return &elasticAgentZIPPackage{
		service: service,
		deploy:  deploy,
	}
}

// AddFiles will add files into the service environment, default destination is /
func (i *elasticAgentZIPPackage) AddFiles(ctx context.Context, files []string) error {
	return nil
}

// Inspect returns info on package
func (i *elasticAgentZIPPackage) Inspect() (deploy.ServiceOperatorManifest, error) {
	return deploy.ServiceOperatorManifest{
		WorkDir:    "C:\\Program Files\\Elastic\\Agent",
		CommitFile: "C:\\elastic-agent\\.elastic-agent.active.commit",
	}, nil
}

// Install installs a package
func (i *elasticAgentZIPPackage) Install(ctx context.Context) error {
	log.Trace("No ZIP install instructions")
	return nil
}

// Exec will execute a command within the service environment
func (i *elasticAgentZIPPackage) Exec(ctx context.Context, args []string) (string, error) {
	span, _ := apm.StartSpanOptions(ctx, "Executing Elastic Agent command", "elastic-agent.zip.exec", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", args)
	defer span.End()

	output, err := i.deploy.ExecIn(ctx, deploy.NewServiceRequest(common.FleetProfileName), i.service, args)
	return output, err
}

// Enroll will enroll the agent into fleet
func (i *elasticAgentZIPPackage) Enroll(ctx context.Context, token string) error {
	cmds := []string{"C:\\elastic-agent\\elastic-agent.exe", "install"}
	span, _ := apm.StartSpanOptions(ctx, "Enrolling Elastic Agent with token", "elastic-agent.zip.enroll", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	defer span.End()

	cfg, _ := kibana.NewFleetConfig(token)
	cmds = append(cmds, cfg.Flags()...)

	_, err := i.Exec(ctx, cmds)
	if err != nil {
		return fmt.Errorf("failed to install the agent with subcommand: %v", err)
	}
	return nil
}

// InstallCerts installs the certificates for a ZIP package, using the right OS package manager
func (i *elasticAgentZIPPackage) InstallCerts(ctx context.Context) error {
	return nil
}

// Logs prints logs of service
func (i *elasticAgentZIPPackage) Logs(ctx context.Context) error {
	// TODO: we need to find a way to read Winidows logs for the service
	// or we could read "C:\Program Files\Elastic\Agent\data\elastic-agent-*\logs\elastic-agent-json.log*"
	return i.deploy.Logs(ctx, i.service)
}

// Postinstall executes operations after installing a ZIP package
func (i *elasticAgentZIPPackage) Postinstall(ctx context.Context) error {
	return nil
}

// Preinstall executes operations before installing a ZIP package
func (i *elasticAgentZIPPackage) Preinstall(ctx context.Context) error {
	span, _ := apm.StartSpanOptions(ctx, "Pre-install operations for the Elastic Agent", "elastic-agent.zip.pre-install", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	artifact := "elastic-agent"
	os := "windows"
	arch := "x86_64"
	extension := "zip"

	_, binaryPath, err := elasticversion.FetchElasticArtifact(ctx, artifact, common.BeatVersion, os, arch, extension, false, true)
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

	output, err := i.Exec(ctx, []string{"powershell.exe", "Test-Path", "C:\\elastic-agent"})
	log.WithFields(log.Fields{
		"output": output,
		"error":  err,
	}).Trace("Checking for existence of elastic-agent installation directory")

	if strings.EqualFold(strings.TrimSpace(output), "false") {
		_, err = i.Exec(ctx, []string{"powershell.exe", "Expand-Archive", "-LiteralPath", binaryPath, "-DestinationPath", "C:\\", "-Force"})
		if err != nil {
			return err
		}

		output, _ := i.Exec(ctx, []string{"powershell.exe", "Move-Item", "-Force", "-Path", fmt.Sprintf("C:\\%s-%s-%s-%s", artifact, elasticversion.GetSnapshotVersion(common.BeatVersion), os, arch), "-Destination", "C:\\elastic-agent"})
		log.WithField("output", output).Trace("Moved elastic-agent")
		return nil
	}

	log.Trace("C:\\elastic-agent already exists, will not attempt to overwrite")
	return nil
}

// Start will start a service
func (i *elasticAgentZIPPackage) Start(ctx context.Context) error {
	return nil
}

// Stop will start a service
func (i *elasticAgentZIPPackage) Stop(ctx context.Context) error {
	return nil
}

// Uninstall uninstalls a EXE package
func (i *elasticAgentZIPPackage) Uninstall(ctx context.Context) error {
	cmds := []string{"C:\\Program Files\\Elastic\\Agent\\elastic-agent.exe", "uninstall", "-f"}
	span, _ := apm.StartSpanOptions(ctx, "Uninstalling Elastic Agent", "elastic-agent.zip.uninstall", apm.SpanOptions{
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
