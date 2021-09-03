// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package action

import (
	"context"
	"runtime"

	"github.com/elastic/e2e-testing/internal/deploy"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// Attach will attach a service operator action to service operator
func Attach(ctx context.Context, deploy deploy.Deployment, service deploy.ServiceRequest, action string, actionOpts map[string]string) (deploy.ServiceOperatorAction, error) {
	span, _ := apm.StartSpanOptions(ctx, "Attaching action to service operator", "action.attach", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	log.WithFields(log.Fields{
		"service":    service,
		"action":     action,
		"actionOpts": actionOpts,
	}).Trace("Attaching action for service")

	switch action {
	case "wait-for-process":
		if runtime.GOOS == "windows" {
			attachAction := AttachActionWaitProcessWin(deploy, service, actionOpts)
			return attachAction, nil
		}
		attachAction := AttachActionWaitProcess(deploy, service, actionOpts)
		return attachAction, nil
	}

	return nil, nil
}
