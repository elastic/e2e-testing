package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
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

// ExecCommandIntoContainer executes a command, as a user, into a container
func ExecCommandIntoContainer(containerName string, user string, cmd []string) error {
	dockerClient := getDockerClient()

	response, err := dockerClient.ContainerExecCreate(
		context.Background(), containerName, types.ExecConfig{
			User:         user,
			Tty:          false,
			AttachStdin:  false,
			AttachStderr: false,
			AttachStdout: false,
			Detach:       true,
			Cmd:          cmd,
		})

	if err != nil {
		return err
	}

	err = dockerClient.ContainerExecStart(
		context.Background(), response.ID, types.ExecStartCheck{
			Detach: true,
			Tty:    false,
		})

	return err
}

// GetDevNetwork returns the developer network, creating it if it does not exist
func GetDevNetwork() (types.NetworkResource, error) {
	dockerClient := getDockerClient()

	ctx := context.Background()

	networkResource, err := dockerClient.NetworkInspect(ctx, networkName, types.NetworkInspectOptions{
		Verbose: true,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"network": networkName,
		}).Warn("Dev Network not found! Creating it now.")

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
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"labels": labelFilters,
		}).Fatal("Cannot list containers")
	}

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
		log.WithFields(log.Fields{
			"error":   err,
			"service": containerName,
		}).Warn("Service could not be removed")

		return err
	}

	log.WithFields(log.Fields{
		"service": containerName,
	}).Info("Service has been removed")

	return nil
}

// RemoveDevNetwork removes the developer network
func RemoveDevNetwork() error {
	dockerClient := getDockerClient()

	ctx := context.Background()

	log.WithFields(log.Fields{
		"network": networkName,
	}).Debug("Removing Dev Network...")

	if err := dockerClient.NetworkRemove(ctx, networkName); err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"network": networkName,
	}).Debug("Dev Network has been removed")

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
		log.WithFields(log.Fields{
			"error":   err,
			"network": networkName,
		}).Fatal("Cannot create Docker Dev Network, which is necessary")
	}

	log.WithFields(log.Fields{
		"network": networkName,
		"id":      response.ID,
	}).Debug("Dev Network has been created")

	return response
}

func getDockerClient() *client.Client {
	if instance != nil {
		return instance
	}

	clientVersion := "1.39"

	instance, err := client.NewClientWithOpts(client.WithVersion(clientVersion))
	if err != nil {
		log.WithFields(log.Fields{
			"error":         err,
			"clientVersion": clientVersion,
		}).Fatal("Cannot get Docker Client")
	}

	return instance
}
