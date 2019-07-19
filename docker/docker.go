package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var instance *client.Client

// InspectContainer returns the JSON representation of the inspection of a
// Docker container, identified by its name
func InspectContainer(name string) (*types.ContainerJSON, error) {
	dockerClient := getDockerClient()

	ctx := context.Background()

	inspect, err := dockerClient.ContainerInspect(ctx, name)
	if err != nil {
		return nil, err
	}

	return &inspect, nil
}

func getDockerClient() *client.Client {
	if instance != nil {
		return instance
	}

	instance, err := client.NewClientWithOpts(client.WithVersion("1.39"))
	if err != nil {
		panic(err)
	}

	return instance
}
