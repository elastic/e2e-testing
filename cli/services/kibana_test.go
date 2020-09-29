package services

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetIntegrations(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, req.URL.String(), ingestManagerIntegrationsURL)

		body := `{"response": [
			{
				"name": "name-1",
				"title": "title-1",
				"version": "version-1",
			},
			{
				"name": "name-2",
				"title": "title-2",
				"version": "version-2",
			}
		]}`
		rw.Write([]byte(body))
	}))
	defer server.Close()

	client := NewKibanaClient().withBaseURL(server.URL)

	_, err := client.GetIntegrations()
	assert.Nil(t, err)
}

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
