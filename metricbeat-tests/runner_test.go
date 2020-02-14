package main

import (
	"flag"
	"os"
	"strconv"
	"testing"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
)

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
	opt.Paths = flag.Args()

	status := godog.RunWithOptions("godog", func(s *godog.Suite) {}, opt)

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}
