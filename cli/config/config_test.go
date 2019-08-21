package config

import (
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/Flaque/filet"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var yaml = []byte(`services:
  liferay:
    bindmounts: {}
    buildbranch: ""
    buildrepository: ""
    containername: ""
    daemon: false
    env: {}
    exposedports:
    - 8080
    image: liferay/portal
    labels: {}
    name: liferay
    networkalias: liferay
    version: "7.0.6-ga7"
`)

var yaml2 = []byte(`services:
  nginx:
    bindmounts: {}
    buildbranch: ""
    buildrepository: ""
    containername: ""
    daemon: false
    env: {}
    exposedports:
    - 80
    image: nginx
    labels: {}
    name: nginx
    networkalias: nginx
    version: "latest"
`)

func TestCheckConfigFileCreatesWorkspaceAtHome(t *testing.T) {
	defer filet.CleanUp(t)

	tmpDir := filet.TmpDir(t, "")

	workspace := path.Join(tmpDir, ".op")

	e, _ := exists(workspace)
	assert.False(t, e)

	checkConfigFile(workspace)

	e, _ = exists(workspace)
	assert.True(t, e)
}

func TestCheckConfigCreatesConfigFileAtHome(t *testing.T) {
	defer filet.CleanUp(t)

	tmpDir := filet.TmpDir(t, "")

	workspace := path.Join(tmpDir, ".op")
	configFile := path.Join(workspace, "config.yml")

	e, _ := exists(configFile)
	assert.False(t, e)

	checkConfigFile(workspace)

	e, _ = exists(configFile)
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

func TestDefaultConfigFileContainsServicesAndStacks(t *testing.T) {
	defer filet.CleanUp(t)

	tmpDir := filet.TmpDir(t, "")

	workspace := path.Join(tmpDir, ".op")

	cfg, _ := readConfig(workspace)

	assert.True(t, (cfg.Services != nil))
	assert.True(t, (cfg.Stacks != nil))
}

func TestMergeConfigurationFromAll(t *testing.T) {
	defer filet.CleanUp(t)

	currentContextPath := path.Join(".", "config.yml")
	ioutil.WriteFile(currentContextPath, yaml, 0644)
	defer func() {
		os.Remove(currentContextPath)
	}()

	envDir := filet.TmpDir(t, "")
	os.Setenv("OP_CONFIG_PATH", envDir)
	ioutil.WriteFile(path.Join(envDir, "config.yml"), yaml2, 0644)

	initTestConfig(t)

	srv, exists := Op.Services["liferay"]
	assert.True(t, exists)
	assert.Equal(t, "liferay/portal", srv.Image)
	assert.Equal(t, []int{8080}, srv.ExposedPorts)
	assert.Equal(t, "liferay", srv.NetworkAlias)
	assert.Equal(t, "7.0.6-ga7", srv.Version)

	srv, exists = Op.Services["nginx"]
	assert.True(t, exists)
	assert.Equal(t, "nginx", srv.Image)
	assert.Equal(t, "nginx", srv.NetworkAlias)
}

func TestMergeConfigurationFromCurrentExecutionContext(t *testing.T) {
	defer filet.CleanUp(t)

	currentContextPath := path.Join(".", "config.yml")
	ioutil.WriteFile(currentContextPath, yaml, 0644)
	defer func() {
		os.Remove(currentContextPath)
	}()

	initTestConfig(t)

	srv, exists := Op.Services["liferay"]
	assert.True(t, exists)
	assert.Equal(t, "liferay/portal", srv.Image)
	assert.Equal(t, []int{8080}, srv.ExposedPorts)
	assert.Equal(t, "liferay", srv.NetworkAlias)
	assert.Equal(t, "7.0.6-ga7", srv.Version)
}

func TestMergeConfigurationFromEnvVariable(t *testing.T) {
	defer filet.CleanUp(t)

	envDir := filet.TmpDir(t, "")
	os.Setenv("OP_CONFIG_PATH", envDir)
	ioutil.WriteFile(path.Join(envDir, "config.yml"), yaml, 0644)

	initTestConfig(t)

	srv, exists := Op.Services["liferay"]
	assert.True(t, exists)
	assert.Equal(t, "liferay/portal", srv.Image)
	assert.Equal(t, []int{8080}, srv.ExposedPorts)
	assert.Equal(t, "liferay", srv.NetworkAlias)
	assert.Equal(t, "7.0.6-ga7", srv.Version)
}

func TestNewConfigPopulatesConfiguration(t *testing.T) {
	defer filet.CleanUp(t)

	initTestConfig(t)

	assert.True(t, (Op.Services != nil))
	assert.True(t, (Op.Stacks != nil))
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
	os.Unsetenv("OP_CONFIG_PATH")
	os.Unsetenv("OP_LOG_LEVEL")
	os.Unsetenv("OP_LOG_INCLUDE_TIMESTAMP")
}

func initTestConfig(t *testing.T) {
	tmpDir := filet.TmpDir(t, "")

	workspace := path.Join(tmpDir, ".op")

	newConfig(workspace)
}
