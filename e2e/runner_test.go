// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package e2e

import (
	"flag"
	"os"
	"path"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	log "github.com/sirupsen/logrus"

	"github.com/elastic/e2e-testing/cli/config"
)

type contextMetadata struct {
	name         string
	modules      []string
	contextFuncs []func(s *godog.Suite) // the functions that hold the steps for a specific
}

func (c *contextMetadata) getFeaturePaths() []string {
	paths := []string{}
	for _, module := range c.modules {
		paths = append(paths, path.Join("features", module, c.name+".feature"))
	}

	return paths
}

var supportedProducts = map[string]*contextMetadata{}

var opt = godog.Options{Output: colors.Colored(os.Stdout)}

func init() {
	config.Init()

	godog.BindFlags("godog.", flag.CommandLine, &opt)
}

func TestMain(m *testing.M) {
	flag.Parse()

	featurePaths, metadatas := parseFeatureFlags(flag.Args())

	if len(metadatas) == 0 {
		log.Error("We did not find anything to execute. Exiting")
		os.Exit(1)
	}

	opt.Paths = featurePaths

	status := godog.RunWithOptions("godog", func(s *godog.Suite) {
		for _, metadata := range metadatas {
			for _, f := range metadata.contextFuncs {
				f(s)
			}
		}
	}, opt)

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

func findSupportedContext(feature string) *contextMetadata {
	for k, ctx := range supportedProducts {
		ctx.name = k // match key with context name
		if k == feature {
			log.WithFields(log.Fields{
				"paths":   ctx.getFeaturePaths(),
				"modules": ctx.modules,
			}).Info("Feature Context found")
			return ctx
		}
	}

	return nil
}

func parseFeatureFlags(flags []string) ([]string, []*contextMetadata) {
	metadatas := []*contextMetadata{}
	featurePaths := []string{}

	if len(flags) == 1 && flags[0] == "all" {
		for k, metadata := range supportedProducts {
			metadata.name = k // match key with context name
			metadatas = append(metadatas, metadata)
		}
	} else {
		for _, feature := range flags {
			metadata := findSupportedContext(feature)

			if metadata == nil {
				log.Warnf("Sorry but we don't support tests for %s at this moment. Skipping it :(", feature)
				continue
			}

			metadatas = append(metadatas, metadata)
			featurePaths = append(featurePaths, metadata.getFeaturePaths()...)
		}
	}

	return featurePaths, metadatas
}
