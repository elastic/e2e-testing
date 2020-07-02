package docker

import (
	"bytes"
	"context"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

var instance *client.Client

// OPNetworkName name of the network used by the tool
const OPNetworkName = "elastic-dev-network"

// ExecCommandIntoContainer executes a command, as a user, into a container
func ExecCommandIntoContainer(ctx context.Context, containerName string, user string, cmd []string) (string, error) {
	dockerClient := getDockerClient()

	detach := false
	tty := false

	log.WithFields(log.Fields{
		"container": containerName,
		"command":   cmd,
		"detach":    detach,
		"tty":       tty,
	}).Debug("Creating command to be executed in container")

	response, err := dockerClient.ContainerExecCreate(
		ctx, containerName, types.ExecConfig{
			User:         user,
			Tty:          tty,
			AttachStdin:  false,
			AttachStderr: true,
			AttachStdout: true,
			Detach:       detach,
			Cmd:          cmd,
		})

	if err != nil {
		log.WithFields(log.Fields{
			"container": containerName,
			"command":   cmd,
			"error":     err,
			"detach":    detach,
			"tty":       tty,
		}).Warn("Could not create command in container")
		return "", err
	}

	log.WithFields(log.Fields{
		"container": containerName,
		"command":   cmd,
		"detach":    detach,
		"tty":       tty,
	}).Debug("Command to be executed in container created")

	resp, err := dockerClient.ContainerExecAttach(ctx, response.ID, types.ExecStartCheck{
		Detach: detach,
		Tty:    tty,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"container": containerName,
			"command":   cmd,
			"detach":    detach,
			"error":     err,
			"tty":       tty,
		}).Error("Could not execute command in container")
		return "", err
	}
	defer resp.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Reader)
	if err != nil {
		log.WithFields(log.Fields{
			"container": containerName,
			"command":   cmd,
			"detach":    detach,
			"error":     err,
			"tty":       tty,
		}).Error("Could not parse command output from container")
		return "", err
	}
	output := buf.String()

	log.WithFields(log.Fields{
		"container": containerName,
		"command":   cmd,
		"detach":    detach,
		"tty":       tty,
	}).Debug("Command sucessfully executed in container")

	output = strings.ReplaceAll(output, "\n", "")

	return output, nil
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

	inspect, err := dockerClient.ContainerInspect(ctx, containers[0].ID)
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
		"network": OPNetworkName,
	}).Debug("Removing Dev Network...")

	if err := dockerClient.NetworkRemove(ctx, OPNetworkName); err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"network": OPNetworkName,
	}).Debug("Dev Network has been removed")

	return nil
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
