package cmd

import (
	"path/filepath"
	"testing"

	"github.com/Flaque/filet"
	io "github.com/elastic/e2e-testing/cli/internal"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

const testResourcesBasePath = "_testresources/"
const dockerComposeMultiple = "docker-compose-multiple.yml"
const dockerComposeSingle = "docker-compose-single.yml"

func TestSanitizeComposeFile_Multiple(t *testing.T) {
	defer filet.CleanUp(t)
	tmpDir := filet.TmpDir(t, "")

	target := filepath.Join(tmpDir, dockerComposeMultiple)
	src := filepath.Join(testResourcesBasePath, dockerComposeMultiple)

	err := sanitizeComposeFile(src, target)
	assert.Nil(t, err)

	bytes, err := io.ReadFile(target)
	assert.Nil(t, err)

	c := compose{}
	err = yaml.Unmarshal(bytes, &c)
	assert.Nil(t, err)

	assert.Equal(t, c.Version, "2.3")
	assert.Equal(t, len(c.Services), 2)
}

func TestSanitizeComposeFile_Single(t *testing.T) {
	defer filet.CleanUp(t)
	tmpDir := filet.TmpDir(t, "")

	target := filepath.Join(tmpDir, dockerComposeSingle)
	src := filepath.Join(testResourcesBasePath, dockerComposeSingle)

	err := sanitizeComposeFile(src, target)
	assert.Nil(t, err)

	bytes, err := io.ReadFile(target)
	assert.Nil(t, err)

	c := compose{}
	err = yaml.Unmarshal(bytes, &c)
	assert.Nil(t, err)

	assert.Equal(t, c.Version, "2.3")
	assert.Equal(t, len(c.Services), 1)
}
