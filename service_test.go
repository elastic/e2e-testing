package main_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	services "github.com/elastic/metricbeat-tests-poc"
)

func TestNewService_ContainerNameIncludesVersionAsSuffix(t *testing.T) {
	apache := services.NewApacheService("2.2", "80")

	assert.Equal(t, "apache-2.2", apache.GetContainerName())
}

func TestNewService_ExposedPortsIsEmpty(t *testing.T) {
	apache := services.NewApacheService("2.2", "80")

	assert.Equal(t, 0, len(apache.GetExposedPorts()))
}

func TestNewService_Name(t *testing.T) {
	apache := services.NewApacheService("2.2", "80")

	assert.Equal(t, "apache", apache.GetName())
}
