package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	"github.com/elastic/metricbeat-tests-poc/log"
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
		log.Info("Dev Network (%s) not found! Creating it now.", networkName)

		initDevNetwork()
	}

	return networkResource, err
}

// InspectContainer returns the JSON representation of the inspection of a
// Docker container, identified by its name
func InspectContainer(name string) (*types.ContainerJSON, error) {
	dockerClient := getDockerClient()

	ctx := context.Background()

	labelFilters := filters.NewArgs()
	labelFilters.Add("label", "service.owner=co.elastic.observability")
	labelFilters.Add("label", "service.container.name="+name)

	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true, Filters: labelFilters})
	log.CheckIfError(err)

	for _, c := range containers {
		inspect, err := dockerClient.ContainerInspect(ctx, c.ID)
		if err != nil {
			return nil, err
		}

		return &inspect, nil
	}

	return nil, nil
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
		log.Warn("Service %s could not be removed: %v", containerName, err)
		return err
	}

	log.Info("Service has been %s removed!", containerName)

	return nil
}

// RemoveDevNetwork removes the developer network
func RemoveDevNetwork() error {
	dockerClient := getDockerClient()

	ctx := context.Background()

	log.Info("Removing Dev Network (%s).", networkName)
	if err := dockerClient.NetworkRemove(ctx, networkName); err != nil {
		return err
	}

	log.Success("Dev Network has been %s removed!", networkName)

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
	log.CheckIfErrorMessage(err, "Cannot create Docker Dev Network which is necessary. Aborting")

	log.Success("Dev Network (%s) has been created with ID %s.", networkName, response.ID)

	return response
}

func getDockerClient() *client.Client {
	if instance != nil {
		return instance
	}

	instance, err := client.NewClientWithOpts(client.WithVersion("1.39"))
	log.CheckIfError(err)

	return instance
}
