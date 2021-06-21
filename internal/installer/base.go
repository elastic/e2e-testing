// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package installer

import (
	"context"
	"strings"

	"github.com/elastic/e2e-testing/internal/deploy"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

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
			install := AttachElasticAgentTARPackage(deploy, service)
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

func systemCtlPostInstall(ctx context.Context, linux string, artifact string, execFn func(ctx context.Context, args []string) (string, error)) error {
	cmds := []string{"systemctl", "restart", artifact}
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
	cmds := []string{"systemctl", "start", artifact}
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
