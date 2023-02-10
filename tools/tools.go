// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build tools
// +build tools

package tools

// Add dependencies on tools.
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
import (
	// Register godog for pinning version
	_ "github.com/cucumber/godog/cmd/godog"
	// Register elastic-package for pinning version
	_ "github.com/elastic/elastic-package"
	// Register gotestsum for pinning version
	_ "gotest.tools/gotestsum"
)
