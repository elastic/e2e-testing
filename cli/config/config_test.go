package config

import (
	"path"
	"testing"

	"github.com/Flaque/filet"
	"github.com/stretchr/testify/assert"
)

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
