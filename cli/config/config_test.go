package config

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/Flaque/filet"
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
	tmpDir := filet.TmpDir(t, "")

	workspace := path.Join(tmpDir, ".op")

	e, _ := exists(workspace)
	assert.False(t, e)

	checkConfigFile(workspace)

	e, _ = exists(workspace)
	assert.True(t, e)
}

func TestCheckConfigCreatesConfigFileAtHome(t *testing.T) {
	tmpDir := filet.TmpDir(t, "")

	workspace := path.Join(tmpDir, ".op")
	configFile := path.Join(workspace, "config.yml")

	e, _ := exists(configFile)
	assert.False(t, e)

	checkConfigFile(workspace)

	e, _ = exists(configFile)
	assert.True(t, e)
}

func TestDefaultConfigFileContainsServicesAndStacks(t *testing.T) {
	tmpDir := filet.TmpDir(t, "")

	workspace := path.Join(tmpDir, ".op")

	cfg, _ := readConfig(workspace)

	assert.True(t, (cfg.Services != nil))
	assert.True(t, (cfg.Stacks != nil))
}

func TestGetStackConfigReturnsAValidStack(t *testing.T) {
	initTestConfig(t)

	stack, exists := GetStackConfig("observability")
	assert.True(t, exists)
	assert.Equal(t, "Observability", stack.Name)
}

func TestGetStackConfigReturnsANonExistingStack(t *testing.T) {
	initTestConfig(t)

	stack, exists := GetStackConfig("foo")

	assert.False(t, exists)
	assert.Equal(t, "", stack.Name)
	assert.Equal(t, 0, len(stack.Services))
}

func TestMergeConfigurationFromAll(t *testing.T) {
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
	initTestConfig(t)

	assert.True(t, (Op.Services != nil))
	assert.True(t, (Op.Stacks != nil))
}

func initTestConfig(t *testing.T) {
	tmpDir := filet.TmpDir(t, "")

	workspace := path.Join(tmpDir, ".op")

	newConfig(workspace)
}
