package services_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	config "github.com/elastic/metricbeat-tests-poc/config"
	services "github.com/elastic/metricbeat-tests-poc/services"
)

var serviceManager services.ServiceManager = services.NewServiceManager()

func TestMain(m *testing.M) {
	config.Init()

	os.Exit(m.Run())
}

func TestBuildService_ContainerNameIncludesVersionAsSuffix(t *testing.T) {
	srv := serviceManager.Build("apache", "2.2", false)

	assert.Equal(t, "apache-2.2", srv.GetContainerName())
}

func TestBuildService_ExposedPortsReturnsDefault(t *testing.T) {
	srv := serviceManager.Build("apache", "2.2", false)

	assert.Equal(t, "80", srv.GetExposedPort())
}

func TestBuildService_Name(t *testing.T) {
	srv := serviceManager.Build("apache", "2.2", false)

	assert.Equal(t, "apache", srv.GetName())
}

func TestNewService_NetworkAlias(t *testing.T) {
	srv := services.DockerService{
		NetworkAlias: "foo",
	}

	assert.Equal(t, "foo", srv.GetNetworkAlias())
}

func TestNewService_NetworkAliasEmptyUsesName(t *testing.T) {
	srv := services.DockerService{
		Name: "name",
	}

	assert.Equal(t, "name", srv.GetNetworkAlias())
}

func TestNewService_NetworkAliasAndNameEmpty(t *testing.T) {
	srv := services.DockerService{}

	assert.Equal(t, "", srv.GetNetworkAlias())
}
