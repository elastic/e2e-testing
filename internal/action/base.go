// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package action

import (
	"context"

	"github.com/elastic/e2e-testing/internal/deploy"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

const (
	// ActionWaitForProcess const for choosing the wait for process action
	ActionWaitForProcess = "wait-for-process"
)

// Attach will attach a service operator action to service operator
func Attach(ctx context.Context, deploy deploy.Deployment, service deploy.ServiceRequest, action string, actionOpts interface{}) (deploy.ServiceOperation, error) {
	span, _ := apm.StartSpanOptions(ctx, "Attaching action to service operator", "action.attach", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	log.WithFields(log.Fields{
		"service": service,
		"action":  action,
	}).Trace("Attaching action for service")

	switch action {
	case ActionWaitForProcess:
		newActionOpts, ok := actionOpts.(ProcessAction)
		if !ok {
			log.Fatal("Unable to cast to action options to ProcessAction")
		}
		attachAction := AttachActionWaitProcess(deploy, service, newActionOpts)
		return attachAction, nil
	}

	log.WithField("action", action).Warn("Unknown action called")
	return nil, nil
}
