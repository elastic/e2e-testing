// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/elastic/e2e-testing/internal/io"

	"github.com/Flaque/filet"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCheckConfigDirsCreatesWorkspaceAtHome(t *testing.T) {
	defer filet.CleanUp(t)

	tmpDir := filet.TmpDir(t, "")

	workspace := path.Join(tmpDir, ".op")

	e, _ := io.Exists(workspace)
	assert.False(t, e)

	checkConfigDirs(workspace)

	e, _ = io.Exists(workspace)
	assert.True(t, e)
	e, _ = io.Exists(path.Join(workspace, "compose", "services"))
	assert.True(t, e)
	e, _ = io.Exists(path.Join(workspace, "compose", "profiles"))
	assert.True(t, e)
}

func TestConfigureLoggerWithTimestamps(t *testing.T) {
	os.Setenv("OP_LOG_INCLUDE_TIMESTAMP", "true")
	defer cleanUpEnv()

	configureLogger()

	logger := logrus.New()

	formatterType := reflect.TypeOf(logger.Formatter)
	assert.Equal(t, "*logrus.TextFormatter", formatterType.String())
}

func TestConfigureLoggerWithDebugLogLevel(t *testing.T) {
	checkLoggerWithLogLevel(t, "DEBUG")
}

func TestConfigureLoggerWithErrorLogLevel(t *testing.T) {
	checkLoggerWithLogLevel(t, "ERROR")
}

func TestConfigureLoggerWithFatalLogLevel(t *testing.T) {
	checkLoggerWithLogLevel(t, "FATAL")
}

func TestConfigureLoggerWithInfoLogLevel(t *testing.T) {
	checkLoggerWithLogLevel(t, "INFO")
}

func TestConfigureLoggerWithPanicLogLevel(t *testing.T) {
	checkLoggerWithLogLevel(t, "PANIC")
}

func TestConfigureLoggerWithTraceLogLevel(t *testing.T) {
	checkLoggerWithLogLevel(t, "TRACE")
}

func TestConfigureLoggerWithWarningLogLevel(t *testing.T) {
	checkLoggerWithLogLevel(t, "WARNING")
}

func TestConfigureLoggerWithWrongLogLevel(t *testing.T) {
	checkLoggerWithLogLevel(t, "FOO_BAR")
}

func TestNewConfigPopulatesConfiguration(t *testing.T) {
	defer filet.CleanUp(t)

	initTestConfig(t)

	assert.True(t, (Op.Services != nil))
	assert.True(t, (Op.Profiles != nil))
}

func checkLoggerWithLogLevel(t *testing.T, level string) {
	os.Setenv("OP_LOG_LEVEL", strings.ToUpper(level))
	defer cleanUpEnv()

	levels := []string{"TRACE", "DEBUG", "INFO", "WARNING", "ERROR", "FATAL", "PANIC"}
	m := make(map[string]bool)
	for _, l := range levels {
		m[l] = true
	}
	if _, exists := m[level]; !exists {
		level = "INFO"
	}

	configureLogger()

	logLevel := logrus.GetLevel()

	assert.Equal(t, strings.ToLower(level), logLevel.String())
}

func cleanUpEnv() {
	os.Unsetenv("OP_LOG_LEVEL")
	os.Unsetenv("OP_LOG_INCLUDE_TIMESTAMP")
}

func initTestConfig(t *testing.T) {
	tmpDir := filet.TmpDir(t, "")

	workspace := path.Join(tmpDir, ".op")

	newConfig(workspace)
}
