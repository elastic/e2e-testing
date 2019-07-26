package services

import (
	"testing"

	services "github.com/elastic/metricbeat-tests-poc/services"
	"github.com/stretchr/testify/assert"
)

func TestNewService_ContainerNameIncludesVersionAsSuffix(t *testing.T) {
	apache := services.NewApacheService("2.2", false)

	assert.Equal(t, "apache-2.2", apache.GetContainerName())
}

func TestNewService_ExposedPortsIsEmpty(t *testing.T) {
	apache := services.NewApacheService("2.2", false)

	assert.Equal(t, "80", apache.GetExposedPort())
}

func TestNewService_Name(t *testing.T) {
	apache := services.NewApacheService("2.2", false)

	assert.Equal(t, "apache", apache.GetName())
}
