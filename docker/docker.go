package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

var instance *client.Client

const networkName = "elastic-dev-network"

// ConnectContainerToDevNetwork connects a container to the Dev Network
func ConnectContainerToDevNetwork(containerID string, aliases ...string) error {
	dockerClient := getDockerClient()

	ctx := context.Background()

	return dockerClient.NetworkConnect(
		ctx, networkName, containerID, &network.EndpointSettings{
			Aliases: aliases,
		})
}

// GetDevNetwork returns the developer network, creating it if it does not exist
func GetDevNetwork() (types.NetworkResource, error) {
	dockerClient := getDockerClient()

	ctx := context.Background()

	networkResource, err := dockerClient.NetworkInspect(ctx, networkName, types.NetworkInspectOptions{
		Verbose: true,
	})
	if err != nil {
		fmt.Printf("Dev Network (%s) not found! Creating it now.\n", networkName)
		initDevNetwork()
	} else {
		fmt.Printf("Dev Network (%s) already exists.\n", networkName)
	}

	return networkResource, err
}

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

// RemoveContainer removes a container identified by its container name
func RemoveContainer(containerName string) error {
	dockerClient := getDockerClient()

	ctx := context.Background()

	options := types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}

	if err := dockerClient.ContainerRemove(ctx, containerName, options); err != nil {
		fmt.Printf("Service %s could not be removed: %v\n", containerName, err)
		return err
	}

	fmt.Printf("Service has been %s removed!\n", containerName)

	return nil
}

// RemoveDevNetwork removes the developer network
func RemoveDevNetwork() error {
	dockerClient := getDockerClient()

	ctx := context.Background()

	fmt.Printf("Removing Dev Network (%s).\n", networkName)
	if err := dockerClient.NetworkRemove(ctx, networkName); err != nil {
		return err
	}

	fmt.Printf("Dev Network has been %s removed!", networkName)

	return nil
}

func initDevNetwork() types.NetworkCreateResponse {
	dockerClient := getDockerClient()

	ctx := context.Background()

	nc := types.NetworkCreate{
		Driver:         "bridge",
		CheckDuplicate: true,
		Internal:       true,
		EnableIPv6:     false,
		Attachable:     true,
		Labels: map[string]string{
			"project": "observability",
		},
	}

	response, err := dockerClient.NetworkCreate(ctx, networkName, nc)
	if err != nil {
		panic("Cannot create Docker Dev Network which is necessary. Aborting: " + err.Error())
	}

	fmt.Printf("Dev Network (%s) has been created with ID %s.\n", networkName, response.ID)

	return response
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
