// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package services

import (
	"encoding/json"
	"strings"

	"github.com/elastic/e2e-testing/cli/shell"
)

// Resource hide the real type of the enum
// and users can use it to define the var for accepting enum
type Resource = string

type resourceList struct {
	ClusterRole        Resource
	ClusterRoleBinding Resource
	ConfigMap          Resource
	Daemonset          Resource
	Deployment         Resource
	ServiceAccount     Resource
}

// ResourceTypes for public use
var ResourceTypes = &resourceList{
	ClusterRole:        "ClusterRole",
	ClusterRoleBinding: "ClusterRoleBinding",
	ConfigMap:          "ConfigMap",
	Daemonset:          "Daemonset",
	Deployment:         "Deployment",
	ServiceAccount:     "ServiceAccount",
}

// Kubectl define some operation using kubectl CLI.
type Kubectl struct {
}

// GetStringResourcesBySelector Use kubectl to get a resource identified by a selector, return the resource in a string.
func (k *Kubectl) GetStringResourcesBySelector(resourceType, selector string) (string, error) {
	output, err := k.Run("get", resourceType, "--selector", selector, "-o", "json")
	if err != nil {
		return "", err
	}

	return output, nil
}

// GetResourcesBySelector Use kubectl to get a resource identified by a selector, return the resource in a map[string]interface{}.
func (k *Kubectl) GetResourcesBySelector(resourceType, selector string) (map[string]interface{}, error) {
	output, err := k.Run("get", resourceType, "--selector", selector, "-o", "json")
	if err != nil {
		return nil, err
	}

	return jsonToObj(output)
}

// GetResourceJSONPath use kubectl to get a resource by type and name, and a jsonpath, and return the definition of the resource in JSON.
func (k *Kubectl) GetResourceJSONPath(resourceType, resource, jsonPath string) (string, error) {
	output, err := k.Run("get", resourceType, resource, "-o", "jsonpath='"+jsonPath+"'")
	if err != nil {
		return "{}", err
	}
	return output, nil
}

// GetResourceSelector use kubectl to get the selector for a resource identified by type and name.
func (k *Kubectl) GetResourceSelector(resourceType, resource string) (string, error) {
	output, err := k.GetResourceJSONPath(resourceType, resource, "{.metadata.selfLink}")
	if err != nil {
		return "", err
	}

	output, err = k.Run("get", "--raw", strings.Replace(output, "'", "", -1)+"/scale")
	if err != nil {
		return "", err
	}

	jsonObj, err := jsonToObj(output)
	if err != nil {
		return "", err
	}

	status := jsonObj["status"].(map[string]interface{})
	return status["selector"].(string), nil
}

// Run a kubectl command and return the output
func (k *Kubectl) Run(args ...string) (string, error) {
	return shell.Execute(".", "kubectl", args...)
}

// jsonToObj Converts a JSON string to a map[string]interface{}.
func jsonToObj(jsonValue string) (map[string]interface{}, error) {
	var obj map[string]interface{}
	err := json.Unmarshal([]byte(jsonValue), &obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}
