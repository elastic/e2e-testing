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
	"github.com/elastic/e2e-testing/internal/systemd"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// ElasticAgentDeployer struct representing the deployer for an elastic-agent
type ElasticAgentDeployer struct {
	deployer deploy.Deployer
}

// NewElasticAgentDeployer retrives a new instance of the elastic-agent deployer
func NewElasticAgentDeployer(deployer deploy.Deployer) ElasticAgentDeployer {
	return ElasticAgentDeployer{
		deployer: deployer,
	}
}

// AttachInstaller will attach an Elastic Agent installer to a deployment allowing
// the installation of a package to be transparently configured no matter the backend
func (ead ElasticAgentDeployer) AttachInstaller(ctx context.Context, service deploy.ServiceRequest, installType string) (deploy.ServiceOperator, error) {
	span, _ := apm.StartSpanOptions(ctx, "Attaching installer to host", "elastic-agent.installer.attach", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	log.WithFields(log.Fields{
		"service":     service,
		"installType": installType,
	}).Trace("Attaching service for configuration")

	deployer := ead.deployer

	if strings.EqualFold(service.Name, "elastic-agent") {
		switch installType {
		case "tar":
			// Since both Linux and macOS distribute elastic-agent using TAR format we must
			// determine the runtime to figure out which tar installer to use here
			if runtime.GOOS == "darwin" && common.Provider == "remote" {
				install := ElasticAgentTARDarwinPackage(deployer, service)
				return install, nil
			}
			install := ElasticAgentTARPackage(deployer, service)
			return install, nil
		case "zip":
			install := ElasticAgentZIPPackage(deployer, service)
			return install, nil
		case "rpm":
			install := ElasticAgentRPMPackage(deployer, service)
			return install, nil
		case "deb":
			install := ElasticAgentDEBPackage(deployer, service)
			return install, nil
		case "docker":
			install := ElasticAgentDockerPackage(deployer, service)
			return install, nil
		}
	}

	return nil, nil
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
