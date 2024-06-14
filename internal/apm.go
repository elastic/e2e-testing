// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"github.com/elastic/e2e-testing/internal/shell"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/module/apmhttp/v2"
)

// StartTransaction returns a new Transaction with the specified
// name and type, with the start time set to the current time and
// with the context if TRACEPARENT environment variable is set.
// This is equivalent to calling apm.DefaultTracer().StartTransaction
// if no TRACEPARENT environment variable otherwise
// apm.DefaultTracer().StartTransactionOptions
func StartTransaction(name, transactionType string) *apm.Transaction {
	traceparent := shell.GetEnv("TRACEPARENT", "")
	if traceparent == "" {
		return apm.DefaultTracer().StartTransaction(name, transactionType)
	}

	traceContext, err := apmhttp.ParseTraceparentHeader(traceparent)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Could not read the traceparent. Fallback to an empty context.")
		return apm.DefaultTracer().StartTransaction(name, transactionType)
	}

	log.WithFields(log.Fields{
		"env.TRACEPARENT": traceparent,
		"traceContext":    apmhttp.FormatTraceparentHeader(traceContext),
	}).Info("Using the given traceparent")

	opts := apm.TransactionOptions{
		TraceContext: traceContext,
	}

	return apm.DefaultTracer().StartTransactionOptions(name, transactionType, opts)
}
