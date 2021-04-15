// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package docker

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/elastic/e2e-testing/internal/common"
	log "github.com/sirupsen/logrus"
)

var instance *client.Client

// OPNetworkName name of the network used by the tool
const OPNetworkName = "elastic-dev-network"

// CheckProcessStateOnTheHost checks if a process is in the desired state in a container
// name of the container for the service:
// we are using the Docker client instead of docker-compose
// because it does not support returning the output of a
// command: it simply returns error level
func CheckProcessStateOnTheHost(containerName string, process string, state string, timeoutFactor int) error {
	timeout := time.Duration(common.TimeoutFactor) * time.Minute

	err := WaitForProcess(containerName, process, state, timeout)
	if err != nil {
		if state == "started" {
			log.WithFields(log.Fields{
				"container ": containerName,
				"error":      err,
				"timeout":    timeout,
			}).Error("The process was not found but should be present")
		} else {
			log.WithFields(log.Fields{
				"container": containerName,
				"error":     err,
				"timeout":   timeout,
			}).Error("The process was found but shouldn't be present")
		}

		return err
	}

	return nil
}

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
	}).Trace("Creating command to be executed in container")

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
	}).Trace("Command to be executed in container created")

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
	}).Trace("Command sucessfully executed in container")

	output = strings.ReplaceAll(output, "\n", "")

	patterns := []string{
		"\x01\x00\x00\x00\x00\x00\x00\r",
		"\x01\x00\x00\x00\x00\x00\x00)",
	}
	for _, pattern := range patterns {
		if strings.HasPrefix(output, pattern) {
			output = strings.ReplaceAll(output, pattern, "")
			log.WithFields(log.Fields{
				"output": output,
			}).Trace("Output name has been sanitized")
		}
	}

	return output, nil
}

// GetContainerHostname we need the container name because we use the Docker Client instead of Docker Compose
func GetContainerHostname(containerName string) (string, error) {
	log.WithFields(log.Fields{
		"containerName": containerName,
	}).Trace("Retrieving container name from the Docker client")

	hostname, err := ExecCommandIntoContainer(context.Background(), containerName, "root", []string{"cat", "/etc/hostname"})
	if err != nil {
		log.WithFields(log.Fields{
			"containerName": containerName,
			"error":         err,
		}).Error("Could not retrieve container name from the Docker client")
		return "", err
	}

	log.WithFields(log.Fields{
		"containerName": containerName,
		"hostname":      hostname,
	}).Info("Hostname retrieved from the Docker client")

	return hostname, nil
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

// LoadImage loads a TAR file in the local docker engine
func LoadImage(imagePath string) error {
	fileNamePath, err := filepath.Abs(imagePath)
	if err != nil {
		return err
	}

	_, err = os.Stat(fileNamePath)
	if err != nil || os.IsNotExist(err) {
		return err
	}

	dockerClient := getDockerClient()
	file, err := os.Open(imagePath)

	input, err := gzip.NewReader(file)
	imageLoadResponse, err := dockerClient.ImageLoad(context.Background(), input, false)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"image": fileNamePath,
		}).Error("Could not load the Docker image.")
		return err
	}

	log.WithFields(log.Fields{
		"image":    fileNamePath,
		"response": imageLoadResponse,
	}).Debug("Docker image loaded successfully")
	return nil
}

// TagImage tags an existing src image into a target one
func TagImage(src string, target string) error {
	dockerClient := getDockerClient()

	maxTimeout := 15 * time.Second
	exp := common.GetExponentialBackOff(maxTimeout)
	retryCount := 0

	tagImageFn := func() error {
		retryCount++

		err := dockerClient.ImageTag(context.Background(), src, target)
		if err != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"src":         src,
				"target":      target,
				"elapsedTime": exp.GetElapsedTime(),
				"retries":     retryCount,
			}).Warn("Could not tag the Docker image.")
			return err
		}

		log.WithFields(log.Fields{
			"src":         src,
			"target":      target,
			"elapsedTime": exp.GetElapsedTime(),
			"retries":     retryCount,
		}).Debug("Docker image tagged successfully")
		return nil
	}

	return backoff.Retry(tagImageFn, exp)
}

// RemoveDevNetwork removes the developer network
func RemoveDevNetwork() error {
	dockerClient := getDockerClient()

	ctx := context.Background()

	log.WithFields(log.Fields{
		"network": OPNetworkName,
	}).Trace("Removing Dev Network...")

	if err := dockerClient.NetworkRemove(ctx, OPNetworkName); err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"network": OPNetworkName,
	}).Trace("Dev Network has been removed")

	return nil
}

// WaitForProcess polls a container executing "ps" command until the process is in the desired state (present or not),
// or a timeout happens
func WaitForProcess(containerName string, process string, desiredState string, maxTimeout time.Duration) error {
	exp := common.GetExponentialBackOff(maxTimeout)

	mustBePresent := false
	if desiredState == "started" {
		mustBePresent = true
	}
	retryCount := 1

	processStatus := func() error {
		log.WithFields(log.Fields{
			"desiredState": desiredState,
			"process":      process,
		}).Trace("Checking process desired state on the container")

		output, err := ExecCommandIntoContainer(context.Background(), containerName, "root", []string{"pgrep", "-n", "-l", process})
		if err != nil {
			log.WithFields(log.Fields{
				"desiredState":  desiredState,
				"elapsedTime":   exp.GetElapsedTime(),
				"error":         err,
				"container":     containerName,
				"mustBePresent": mustBePresent,
				"process":       process,
				"retry":         retryCount,
			}).Warn("Could not execute 'pgrep -n -l' in the container")

			retryCount++

			return err
		}

		outputContainsProcess := strings.Contains(output, process)

		// both true or both false
		if mustBePresent == outputContainsProcess {
			log.WithFields(log.Fields{
				"desiredState":  desiredState,
				"container":     containerName,
				"mustBePresent": mustBePresent,
				"process":       process,
			}).Infof("Process desired state checked")

			return nil
		}

		if mustBePresent {
			err = fmt.Errorf("%s process is not running in the container yet", process)
			log.WithFields(log.Fields{
				"desiredState": desiredState,
				"elapsedTime":  exp.GetElapsedTime(),
				"error":        err,
				"container":    containerName,
				"process":      process,
				"retry":        retryCount,
			}).Warn(err.Error())

			retryCount++

			return err
		}

		err = fmt.Errorf("%s process is still running in the container", process)
		log.WithFields(log.Fields{
			"elapsedTime": exp.GetElapsedTime(),
			"error":       err,
			"container":   containerName,
			"process":     process,
			"state":       desiredState,
			"retry":       retryCount,
		}).Warn(err.Error())

		retryCount++

		return err
	}

	err := backoff.Retry(processStatus, exp)
	if err != nil {
		return err
	}

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
