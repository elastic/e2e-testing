package cmd

import (
	"fmt"
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

	assert.Equal(t, c.Version, "2.4")
	assert.Equal(t, len(c.Services), 2)

	// we know that both services have different number of ports
	for k, srv := range c.Services {
		switch i := srv.(type) {
		case map[interface{}]interface{}:
			for key, value := range i {
				strKey := fmt.Sprintf("%v", key)

				// does not contain the build context element
				assert.NotEqual(t, strKey, "build")

				// strKey == ports
				if strKey == "ports" {
					if k == "ceph" {
						// ceph has 3 ports
						assert.Equal(t, len(value.([]interface{})), 3)
					} else if k == "ceph-api" {
						// ceph-api has 1 port
						assert.Equal(t, len(value.([]interface{})), 1)
					}
				}
			}
		}
	}
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

	assert.Equal(t, c.Version, "2.4")
	assert.Equal(t, len(c.Services), 1)
}
