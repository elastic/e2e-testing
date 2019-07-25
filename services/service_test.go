package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewService_ContainerNameIncludesVersionAsSuffix(t *testing.T) {
	apache := NewApacheService("2.2", false)

	assert.Equal(t, "apache-2.2", apache.GetContainerName())
}

func TestNewService_ExposedPortsIsEmpty(t *testing.T) {
	apache := NewApacheService("2.2", false)

	assert.Equal(t, 80, apache.GetExposedPort())
}

func TestNewService_Name(t *testing.T) {
	apache := NewApacheService("2.2", false)

	assert.Equal(t, "apache", apache.GetName())
}
