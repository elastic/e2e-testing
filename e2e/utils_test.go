package e2e

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadFile(t *testing.T) {
	configurationFileURL := "https://raw.githubusercontent.com/elastic/beats/v7.6.0/metricbeat/metricbeat.yml"

	configurationFilePath, err := downloadFile(configurationFileURL)
	if err != nil {
		t.Fail()
	}
	defer os.Remove(configurationFilePath)

	info, err := os.Stat(configurationFilePath)
	if os.IsNotExist(err) {
		t.Fail()
	}

	assert.False(t, info.IsDir())
	assert.True(t, strings.HasPrefix(info.Name(), "metricbeat.yml"))
}
