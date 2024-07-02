// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/io"
	"github.com/elastic/e2e-testing/internal/systemd"
	"github.com/elastic/e2e-testing/pkg/downloads"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm/v2"
)

type elasticAgentPackage struct {
	service  deploy.ServiceRequest
	deploy   deploy.Deployment
	metadata deploy.ServiceInstallerMetadata
}

// Metadata returns the type of the package
func (p *elasticAgentPackage) PkgMetadata() deploy.ServiceInstallerMetadata {
	return p.metadata
}

// Attach will attach a installer to a deployment allowing
// the installation of a package to be transparently configured no matter the backend
func Attach(ctx context.Context, deploy deploy.Deployment, service deploy.ServiceRequest, installType string) (deploy.ServiceOperator, error) {
	span, _ := apm.StartSpanOptions(ctx, "Attaching installer to host", "elastic-agent.installer.attach", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	log.WithFields(log.Fields{
		"service":     service,
		"installType": installType,
	}).Trace("Attaching service for configuration")

	if strings.EqualFold(service.Name, "elastic-agent") {
		switch installType {
		case "tar":
			// Since both Linux and macOS distribute elastic-agent using TAR format we must
			// determine the runtime to figure out which tar installer to use here
			if runtime.GOOS == "darwin" && common.Provider == "remote" {
				install := AttachElasticAgentTARDarwinPackage(deploy, service)
				return install, nil
			}
			install := AttachElasticAgentTARPackage(deploy, service)
			return install, nil
		case "zip":
			install := AttachElasticAgentZIPPackage(deploy, service)
			return install, nil
		case "rpm":
			install := AttachElasticAgentRPMPackage(deploy, service)
			return install, nil
		case "deb":
			install := AttachElasticAgentDEBPackage(deploy, service)
			return install, nil
		case "docker":
			install := AttachElasticAgentDockerPackage(deploy, service)
			return install, nil
		}
	}

	return nil, nil
}

// doUpgrade upgrade an elastic-agent package using the 'upgrade' command
func doUpgrade(ctx context.Context, so deploy.ServiceOperator) error {
	pkgMetadata := so.PkgMetadata()

	// downloading target release for the upgrade
	version := common.ElasticAgentVersion

	artifact := common.ElasticAgentServiceName
	_, binaryPath, err := downloads.FetchElasticArtifactForSnapshots(ctx, false, artifact, version, pkgMetadata.Os, pkgMetadata.Arch, pkgMetadata.FileExtension, pkgMetadata.Docker, pkgMetadata.XPack)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact":        artifact,
			"version":         version,
			"packageMetadata": pkgMetadata,
			"error":           err,
		}).Error("Could not download the binary for the agent")
		return err
	}

	if downloads.SnapshotHasCommit(version) {
		version = downloads.RemoveCommitFromSnapshot(version)
	}

	cmds := []string{"elastic-agent", "upgrade", version, "-v"}
	if pkgMetadata.PackageType == "zip" {
		cmds = []string{`C:\Program Files\Elastic\Agent\elastic-agent.exe`, "uninstall", version, "-v"}
	}
	cmds = append(cmds, "--source-uri", "file://"+binaryPath)

	span, _ := apm.StartSpanOptions(ctx, "Upgrading Elastic Agent", "elastic-agent."+pkgMetadata.PackageType+".upgrade", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	span.Context.SetLabel("runtime", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
	defer span.End()

	_, err = so.Exec(ctx, cmds)
	if err != nil {
		return fmt.Errorf("failed to upgrade the agent with subcommand: %v", err)
	}
	return nil
}

// createAgentDirectories makes sure the agent directories belong to the root user
func createAgentDirectories(ctx context.Context, i deploy.ServiceOperator, osArgs []string) error {
	agentPath := i.PkgMetadata().AgentPath

	err := io.MkdirAll(agentPath)
	if err != nil {
		return err
	}

	output, err := i.Exec(ctx, osArgs)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"args":   osArgs,
		"output": output,
		"path":   agentPath,
	}).Debug("Agent directories will belong to root")
	return nil
}

func systemCtlLog(ctx context.Context, OS string, execFn func(ctx context.Context, args []string) (string, error)) error {
	cmds := systemd.LogCmds(common.ElasticAgentServiceName)
	span, _ := apm.StartSpanOptions(ctx, "Retrieving logs for the Elastic Agent service", "elastic-agent."+OS+".log", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	defer span.End()

	logs, err := execFn(ctx, cmds)
	if err != nil {
		return err
	}

	// print logs as is, including tabs and line breaks
	fmt.Println(logs)

	return nil
}

func systemCtlPostInstall(ctx context.Context, linux string, artifact string, execFn func(ctx context.Context, args []string) (string, error)) error {
	cmds := systemd.RestartCmds(artifact)
	span, _ := apm.StartSpanOptions(ctx, "Post-install operations for the "+artifact, artifact+"."+linux+".post-install", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	span.Context.SetLabel("artifact", artifact)
	span.Context.SetLabel("linux", linux)
	defer span.End()

	_, err := execFn(ctx, cmds)
	if err != nil {
		return err
	}
	return nil
}

func systemCtlRestart(ctx context.Context, linux string, artifact string, execFn func(ctx context.Context, args []string) (string, error)) error {
	cmds := systemd.RestartCmds(artifact)
	span, _ := apm.StartSpanOptions(ctx, "Restarting "+artifact+" service", artifact+"."+linux+".restart", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	span.Context.SetLabel("artifact", artifact)
	span.Context.SetLabel("linux", linux)
	defer span.End()

	_, err := execFn(ctx, cmds)
	if err != nil {
		return err
	}
	return nil
}

func systemCtlStart(ctx context.Context, linux string, artifact string, execFn func(ctx context.Context, args []string) (string, error)) error {
	cmds := systemd.StartCmds(artifact)
	span, _ := apm.StartSpanOptions(ctx, "Starting "+artifact+" service", artifact+"."+linux+".start", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("arguments", cmds)
	span.Context.SetLabel("artifact", artifact)
	span.Context.SetLabel("linux", linux)
	defer span.End()

	_, err := execFn(ctx, cmds)
	if err != nil {
		return err
	}

	return nil
}
