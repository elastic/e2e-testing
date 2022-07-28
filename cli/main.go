// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"github.com/elastic/e2e-testing/cli/cmd"
	"github.com/elastic/e2e-testing/internal/config"
)

func init() {
	config.Init()
}

func main() {
	cmd.Execute()
}
