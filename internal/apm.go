// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"github.com/elastic/e2e-testing/internal/shell"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// StartTransaction returns a new Transaction with the specified
// name and type, with the start time set to the current time and
// with the context if TRACEPARENT environment variable is set.
// This is equivalent to calling apm.DefaultTracer.StartTransaction
// if no TRACEPARENT environment variable otherwise
// apm.DefaultTracer.StartTransactionOptions
func StartTransaction(name, transactionType string) *apm.Transaction {
	traceparent := shell.GetEnv("TRACEPARENT", "")
	if traceparent != "" {
		log.WithFields(log.Fields{
			"traceparent": traceparent,
		}).Debug("Using the given traceparent")
		return apm.DefaultTracer.StartTransaction(name, transactionType)
	}

	return apm.DefaultTracer.StartTransaction(name, transactionType)
}
