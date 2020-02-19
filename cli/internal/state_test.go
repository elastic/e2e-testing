package internal

import (
	"path/filepath"
	"testing"

	"github.com/Flaque/filet"
	"github.com/stretchr/testify/assert"
)

func TestRecover(t *testing.T) {
	defer filet.CleanUp(t)

	tmpDir := filet.TmpDir(t, "")

	workspace := filepath.Join(tmpDir, ".op")

	ID := "mystack-stack"
	composeFiles := []string{
		filepath.Join(workspace, "compose/services/a/1.yml"),
		filepath.Join(workspace, "compose/services/b/2.yml"),
		filepath.Join(workspace, "compose/services/c/3.yml"),
		filepath.Join(workspace, "compose/services/d/4.yml"),
	}
	initialEnv := map[string]string{
		"foo": "bar",
	}

	_ = MkdirAll(workspace)

	Update(ID, workspace, composeFiles, initialEnv)

	runFile := filepath.Join(workspace, ID+".run")
	e, _ := Exists(runFile)
	assert.True(t, e)

	env := Recover(ID, workspace)

	value, e := env["foo"]
	assert.True(t, e)
	assert.Equal(t, "bar", value)
}

func TestUpdateCreatesStateFile(t *testing.T) {
	defer filet.CleanUp(t)

	tmpDir := filet.TmpDir(t, "")

	workspace := filepath.Join(tmpDir, ".op")

	ID := "mystack-stack"
	composeFiles := []string{
		filepath.Join(workspace, "compose/services/a/1.yml"),
		filepath.Join(workspace, "compose/services/b/2.yml"),
		filepath.Join(workspace, "compose/services/c/3.yml"),
		filepath.Join(workspace, "compose/services/d/4.yml"),
	}
	runFile := filepath.Join(workspace, ID+".run")
	_ = MkdirAll(runFile)

	Update(ID, workspace, composeFiles, map[string]string{})

	e, _ := Exists(runFile)
	assert.True(t, e)
}
