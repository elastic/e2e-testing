package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	client := NewKibanaClient()

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:5601", client.getURL())
}

func TestNewKibanaClientWithPathStartingWithSlash(t *testing.T) {
	client := NewKibanaClient().withURL("/with_slash")
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:5601/with_slash", client.getURL())
}

func TestNewKibanaClientWithPathStartingWithoutSlash(t *testing.T) {
	client := NewKibanaClient().withURL("without_slash")
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:5601/without_slash", client.getURL())
}

func TestNewKibanaClientWithMultiplePathsKeepsLastOne(t *testing.T) {
	client := NewKibanaClient().withURL("/with_slash").withURL("lastOne")
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:5601/lastOne", client.getURL())
}
