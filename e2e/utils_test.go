package e2e

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadFile(t *testing.T) {
	configurationFileURL := "https://raw.githubusercontent.com/elastic/beats/v7.6.0/metricbeat/metricbeat.yml"

	configurationFilePath, err := downloadFile(configurationFileURL)
	if err != nil {
		t.Fail()
	}

	info, err := os.Stat(configurationFilePath)
	if os.IsNotExist(err) {
		t.Fail()
	}

	assert.False(t, info.IsDir())
	assert.Equal(t, "metricbeat.yml", info.Name())
}
