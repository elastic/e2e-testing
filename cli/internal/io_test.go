package internal

import (
	"path"
	"testing"

	"github.com/Flaque/filet"
	"github.com/stretchr/testify/assert"
)

func TestMkdirAll(t *testing.T) {
	defer filet.CleanUp(t)

	tmpDir := filet.TmpDir(t, "")

	dir := path.Join(tmpDir, ".op", "compose", "services")

	err := MkdirAll(dir)
	assert.Nil(t, err)

	e, _ := Exists(dir)
	assert.True(t, e)
}
