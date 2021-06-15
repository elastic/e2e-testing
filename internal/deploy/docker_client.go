// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
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

// CopyFileToContainer copies a file to the running container
func CopyFileToContainer(ctx context.Context, containerName string, srcPath string, parentDir string, isTar bool) error {
	dockerClient := getDockerClient()
	defer dockerClient.Close()

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
func ExecCommandIntoContainer(ctx context.Context, container string, user string, cmd []string) (string, error) {
	return ExecCommandIntoContainerWithEnv(ctx, container, user, cmd, []string{})
}

// ExecCommandIntoContainerWithEnv executes a command, as a user, with env, into a container
func ExecCommandIntoContainerWithEnv(ctx context.Context, container string, user string, cmd []string, env []string) (string, error) {
	dockerClient := getDockerClient()
	defer dockerClient.Close()

	detach := false
	tty := false

	containerName := container

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
	defer resp.Close()

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
func InspectContainer(service ServiceRequest) (*types.ContainerJSON, error) {
	dockerClient := getDockerClient()
	defer dockerClient.Close()

	ctx := context.Background()

	labelFilters := filters.NewArgs()
	labelFilters.Add("name", service.Name)

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

// ListContainers returns a list of running containers
func ListContainers() ([]types.Container, error) {
	dockerClient := getDockerClient()
	defer dockerClient.Close()
	ctx := context.Background()

	containers, err := dockerClient.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return []types.Container{}, err
	}
	return containers, nil
}

// RemoveContainer removes a container identified by its container name
func RemoveContainer(containerName string) error {
	dockerClient := getDockerClient()
	defer dockerClient.Close()
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
	defer dockerClient.Close()
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
	defer dockerClient.Close()
	maxTimeout := 5 * time.Second * time.Duration(utils.TimeoutFactor)
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
	defer dockerClient.Close()
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

func getDockerClient() *client.Client {
	if instance != nil {
		return instance
	}

	var clientOpts []client.Opt

	clientVersion := "1.39"

	clientOpts = append(clientOpts, client.WithVersion(clientVersion))

	dockerHost := shell.GetEnv("DOCKER_HOST", "")
	if dockerHost != "" {
		helper, err := connhelper.GetConnectionHelper(dockerHost)
		if err != nil {
			log.Fatal("Could not parse DOCKER_HOST")
		}

		httpClient := &http.Client{
			// No tls
			// No proxy
			Transport: &http.Transport{
				DialContext: helper.Dialer,
			},
		}
		clientOpts = append(clientOpts, client.WithHost(helper.Host), client.WithHTTPClient(httpClient), client.WithDialContext(helper.Dialer))
	}

	instance, err := client.NewClientWithOpts(clientOpts...)
	if err != nil {
		log.WithFields(log.Fields{
			"error":         err,
			"clientVersion": clientVersion,
		}).Fatal("Cannot get Docker Client")
	}

	return instance
}

// PullImages pulls images
func PullImages(ctx context.Context, images []string) error {
	span, _ := apm.StartSpanOptions(ctx, "Pulling images using Docker client", "docker.images.pull", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	c := getDockerClient()
	defer c.Close()

	platform := "linux/" + utils.GetArchitecture()

	log.WithFields(log.Fields{
		"images":   images,
		"platform": platform,
	}).Info("Pulling Docker images...")
	options := types.ImagePullOptions{
		Platform: platform,
	}

	for _, image := range images {
		r, err := c.ImagePull(ctx, image, options)
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
