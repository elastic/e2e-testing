// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

var instance *client.Client

// OPNetworkName name of the network used by the tool
const OPNetworkName = "elastic-dev-network"

type execResult struct {
	StdOut   string
	StdErr   string
	ExitCode int
}

func buildTarForDeployment(file *os.File) (bytes.Buffer, error) {
	fileInfo, _ := file.Stat()

	var buffer bytes.Buffer
	tarWriter := tar.NewWriter(&buffer)
	err := tarWriter.WriteHeader(&tar.Header{
		Name: fileInfo.Name(),
		Mode: 0777,
		Size: int64(fileInfo.Size()),
	})
	if err != nil {
		log.WithFields(log.Fields{
			"fileInfoName": fileInfo.Name(),
			"size":         fileInfo.Size(),
			"error":        err,
		}).Error("Could not build TAR header")
		return bytes.Buffer{}, fmt.Errorf("could not build TAR header: %v", err)
	}

	b, err := ioutil.ReadFile(file.Name())
	if err != nil {
		return bytes.Buffer{}, err
	}

	tarWriter.Write(b)
	defer tarWriter.Close()

	return buffer, nil
}

// CheckProcessStateOnTheHost checks if a process is in the desired state in a container
// name of the container for the service:
// we are using the Docker client instead of docker-compose
// because it does not support returning the output of a
// command: it simply returns error level
func CheckProcessStateOnTheHost(containerName string, process string, state string, occurrences int, timeoutFactor int) error {
	timeout := time.Duration(common.TimeoutFactor) * time.Minute

	err := WaitForProcess(containerName, process, state, occurrences, timeout)
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

// CopyFileToContainer copies a file to the running container
func CopyFileToContainer(ctx context.Context, containerName string, srcPath string, parentDir string, isTar bool) error {
	dockerClient := getDockerClient()

	log.WithFields(log.Fields{
		"container": containerName,
		"src":       srcPath,
		"parentDir": parentDir,
	}).Trace("Copying file to container")

	targetDirectory := filepath.Dir(parentDir)

	_, err := dockerClient.ContainerStatPath(ctx, containerName, targetDirectory)
	if err != nil {
		log.WithFields(log.Fields{
			"container": containerName,
			"error":     err,
			"src":       srcPath,
			"target":    targetDirectory,
		}).Error("Could not get parent directory in the container")
		return err
	}

	file, err := os.Open(srcPath)
	if err != nil {
		log.WithFields(log.Fields{
			"container": containerName,
			"error":     err,
			"src":       srcPath,
			"parentDir": parentDir,
		}).Error("Could not open file to deploy")
		return err
	}
	defer file.Close()

	// TODO: detect the file has TAR headers
	var buffer bytes.Buffer
	if !isTar {
		buffer, err = buildTarForDeployment(file)
		if err != nil {
			return err
		}
	} else {
		writer := bufio.NewWriter(&buffer)
		b, err := ioutil.ReadFile(file.Name())
		if err != nil {
			return err
		}

		writer.Write(b)
	}

	err = dockerClient.CopyToContainer(ctx, containerName, parentDir, &buffer, types.CopyToContainerOptions{AllowOverwriteDirWithFile: true})
	if err != nil {
		log.WithFields(log.Fields{
			"container": containerName,
			"error":     err,
			"src":       srcPath,
			"parentDir": parentDir,
		}).Error("Could not copy file to container")
		return err
	}

	return nil
}

// ExecCommandIntoContainer executes a command, as a user, into a container
func ExecCommandIntoContainer(ctx context.Context, containerName string, user string, cmd []string) (string, error) {
	return ExecCommandIntoContainerWithEnv(ctx, containerName, user, cmd, []string{})
}

// ExecCommandIntoContainerWithEnv executes a command, as a user, with env, into a container
func ExecCommandIntoContainerWithEnv(ctx context.Context, containerName string, user string, cmd []string, env []string) (string, error) {
	dockerClient := getDockerClient()

	detach := false
	tty := false

	log.WithFields(log.Fields{
		"container": containerName,
		"command":   cmd,
		"detach":    detach,
		"env":       env,
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
			Env:          env,
		})

	if err != nil {
		log.WithFields(log.Fields{
			"container": containerName,
			"command":   cmd,
			"env":       env,
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
		"env":       env,
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
			"env":       env,
			"error":     err,
			"tty":       tty,
		}).Error("Could not execute command in container")
		return "", err
	}
	defer resp.Close()

	// see https://stackoverflow.com/a/57132902
	var execRes execResult

	// read the output
	var outBuf, errBuf bytes.Buffer
	outputDone := make(chan error)

	go func() {
		// StdCopy demultiplexes the stream into two buffers
		_, err = stdcopy.StdCopy(&outBuf, &errBuf, resp.Reader)
		outputDone <- err
	}()

	select {
	case err := <-outputDone:
		if err != nil {
			return "", err
		}
		break

	case <-ctx.Done():
		return "", ctx.Err()
	}

	stdout, err := ioutil.ReadAll(&outBuf)
	if err != nil {
		log.WithFields(log.Fields{
			"container": containerName,
			"command":   cmd,
			"detach":    detach,
			"env":       env,
			"error":     err,
			"tty":       tty,
		}).Error("Could not parse stdout from container")
		return "", err
	}
	stderr, err := ioutil.ReadAll(&errBuf)
	if err != nil {
		log.WithFields(log.Fields{
			"container": containerName,
			"command":   cmd,
			"detach":    detach,
			"env":       env,
			"error":     err,
			"tty":       tty,
		}).Error("Could not parse stderr from container")
		return "", err
	}

	execRes.ExitCode = 0
	execRes.StdOut = string(stdout)
	execRes.StdErr = string(stderr)

	// remove '\n' from the response
	return strings.ReplaceAll(execRes.StdOut, "\n", ""), nil
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
	exp := utils.GetExponentialBackOff(maxTimeout)
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
func WaitForProcess(containerName string, process string, desiredState string, ocurrences int, maxTimeout time.Duration) error {
	exp := common.GetExponentialBackOff(maxTimeout)

	mustBePresent := false
	if desiredState == "started" {
		mustBePresent = true
	}
	retryCount := 1

	processStatus := func() error {
		log.WithFields(log.Fields{
			"desiredState": desiredState,
			"ocurrences":   ocurrences,
			"process":      process,
		}).Trace("Checking process desired state on the container")

		// pgrep -d: -d, --delimiter <string>  specify output delimiter
		//i.e. "pgrep -d , metricbeat": 483,519
		cmds := []string{"pgrep", "-d", ",", process}
		output, err := ExecCommandIntoContainer(context.Background(), containerName, "root", cmds)
		if err != nil {
			log.WithFields(log.Fields{
				"cmds":          cmds,
				"desiredState":  desiredState,
				"elapsedTime":   exp.GetElapsedTime(),
				"error":         err,
				"container":     containerName,
				"mustBePresent": mustBePresent,
				"ocurrences":    ocurrences,
				"process":       process,
				"retry":         retryCount,
			}).Warn("Could not get number of processes in the container")

			retryCount++

			return err
		}

		// tokenize the pids to get each pid's state, adding them to an array if they match the desired state
		// From Split docs:
		// If output does not contain sep and sep is not empty, Split returns a
		// slice of length 1 whose only element is s, that's why we first initialise to the empty array
		pids := strings.Split(output, ",")
		if len(pids) == 1 && pids[0] == "" {
			pids = []string{}
		}

		log.WithFields(log.Fields{
			"count":         len(pids),
			"desiredState":  desiredState,
			"mustBePresent": mustBePresent,
			"pids":          pids,
			"process":       process,
		}).Tracef("Pids for process found")

		desiredStatePids := []string{}

		for _, pid := range pids {
			pidStateCmds := []string{"ps", "-q", pid, "-o", "state", "--no-headers"}
			pidState, err := ExecCommandIntoContainer(context.Background(), containerName, "root", pidStateCmds)
			if err != nil {
				log.WithFields(log.Fields{
					"cmds":          cmds,
					"desiredState":  desiredState,
					"elapsedTime":   exp.GetElapsedTime(),
					"error":         err,
					"container":     containerName,
					"mustBePresent": mustBePresent,
					"ocurrences":    ocurrences,
					"pid":           pid,
					"process":       process,
					"retry":         retryCount,
				}).Warn("Could not check pid status in the container")

				retryCount++

				return err
			}

			log.WithFields(log.Fields{
				"desiredState":  desiredState,
				"mustBePresent": mustBePresent,
				"pid":           pid,
				"pidState":      pidState,
				"process":       process,
			}).Tracef("Checking if process is in the S state")

			// if the process must be present, then check for the S state
			// From 'man ps':
			// D    uninterruptible sleep (usually IO)
			// R    running or runnable (on run queue)
			// S    interruptible sleep (waiting for an event to complete)
			// T    stopped by job control signal
			// t    stopped by debugger during the tracing
			// W    paging (not valid since the 2.6.xx kernel)
			// X    dead (should never be seen)
			// Z    defunct ("zombie") process, terminated but not reaped by its parent
			if mustBePresent && pidState == "S" {
				desiredStatePids = append(desiredStatePids, pid)
			} else if !mustBePresent {
				desiredStatePids = append(desiredStatePids, pid)
			}
		}

		occurrencesMatched := (len(desiredStatePids) == ocurrences)

		// both true or both false
		if mustBePresent == occurrencesMatched {
			log.WithFields(log.Fields{
				"desiredOcurrences": ocurrences,
				"desiredState":      desiredState,
				"container":         containerName,
				"mustBePresent":     mustBePresent,
				"ocurrences":        len(desiredStatePids),
				"process":           process,
			}).Infof("Process desired state checked")

			return nil
		}

		if mustBePresent {
			err = fmt.Errorf("%s process is not running in the container with the desired number of occurrences (%d) yet", process, ocurrences)
			log.WithFields(log.Fields{
				"desiredOcurrences": ocurrences,
				"desiredState":      desiredState,
				"elapsedTime":       exp.GetElapsedTime(),
				"error":             err,
				"container":         containerName,
				"ocurrences":        len(desiredStatePids),
				"process":           process,
				"retry":             retryCount,
			}).Warn(err.Error())

			retryCount++

			return err
		}

		err = fmt.Errorf("%s process is still running in the container", process)
		log.WithFields(log.Fields{
			"desiredOcurrences": ocurrences,
			"elapsedTime":       exp.GetElapsedTime(),
			"error":             err,
			"container":         containerName,
			"ocurrences":        len(desiredStatePids),
			"process":           process,
			"state":             desiredState,
			"retry":             retryCount,
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

// PullImages pulls images
func PullImages(images []string) error {
	c := getDockerClient()
	ctx := context.Background()

	log.WithField("images", images).Info("Pulling Docker images...")
	for _, image := range images {
		r, err := c.ImagePull(ctx, image, types.ImagePullOptions{})
		if err != nil {
			return err
		}
		_, err = io.Copy(os.Stdout, r)
		if err != nil {
			return err
		}
	}
	return nil
}
