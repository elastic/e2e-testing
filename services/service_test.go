package services_test

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

func TestNewService_NetworkAlias(t *testing.T) {
	service := services.DockerService{
		NetworkAlias: "foo",
	}

	assert.Equal(t, "foo", service.GetNetworkAlias())
}

func TestNewService_NetworkAliasEmptyUsesName(t *testing.T) {
	service := services.DockerService{
		Name: "name",
	}

	assert.Equal(t, "name", service.GetNetworkAlias())
}

func TestNewService_NetworkAliasAndNameEmpty(t *testing.T) {
	service := services.DockerService{}

	assert.Equal(t, "", service.GetNetworkAlias())
}
