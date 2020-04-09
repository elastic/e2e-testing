package e2e

import (
	"flag"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	log "github.com/sirupsen/logrus"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
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

var supportedProducts = map[string]*contextMetadata{
	"apache": &contextMetadata{
		contextFuncs: []func(s *godog.Suite){
			MetricbeatFeatureContext,
		},
		modules: []string{"metricbeat"},
	},
	"helm": &contextMetadata{
		contextFuncs: []func(s *godog.Suite){
			HelmChartFeatureContext,
		},
		modules: []string{"apm", "filebeat", "metricbeat"},
	},
	"filebeat": &contextMetadata{
		contextFuncs: []func(s *godog.Suite){
			HelmChartFeatureContext,
		},
		modules: []string{"filebeat"},
	},
	"metricbeat": &contextMetadata{
		contextFuncs: []func(s *godog.Suite){
			MetricbeatFeatureContext,
		},
		modules: []string{"metricbeat"},
	},
	"mysql": &contextMetadata{
		contextFuncs: []func(s *godog.Suite){
			MetricbeatFeatureContext,
		},
		modules: []string{"metricbeat"},
	},
	"redis": &contextMetadata{
		contextFuncs: []func(s *godog.Suite){
			MetricbeatFeatureContext,
		},
		modules: []string{"metricbeat"},
	},
	"vsphere": &contextMetadata{
		contextFuncs: []func(s *godog.Suite){
			MetricbeatFeatureContext,
		},
		modules: []string{"metricbeat"},
	},
}

// stackVersion is the version of the stack to use
// It can be overriden by OP_STACK_VERSION env var
var stackVersion = "7.6.0"

var opt = godog.Options{Output: colors.Colored(os.Stdout)}

// queryMaxAttempts is the number of attempts to query elasticsearch before aborting
// It can be overriden by OP_QUERY_MAX_ATTEMPTS env var
var queryMaxAttempts = 5

// queryRetryTimeout is the number of seconds between elasticsearch retry queries.
// It can be overriden by OP_RETRY_TIMEOUT env var
var queryRetryTimeout = 3

func getEnv(envVar string, defaultValue string) string {
	if value, exists := os.LookupEnv(envVar); exists {
		return value
	}

	return defaultValue
}

func getIntegerFromEnv(envVar string, defaultValue int) int {
	if value, exists := os.LookupEnv(envVar); exists {
		v, err := strconv.Atoi(value)
		if err == nil {
			return v
		}
	}

	return defaultValue
}

func init() {
	config.Init()

	godog.BindFlags("godog.", flag.CommandLine, &opt)

	stackVersion = getEnv("OP_STACK_VERSION", stackVersion)

	queryMaxAttempts = getIntegerFromEnv("OP_QUERY_MAX_ATTEMPTS", queryMaxAttempts)
	queryRetryTimeout = getIntegerFromEnv("OP_RETRY_TIMEOUT", queryRetryTimeout)
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
