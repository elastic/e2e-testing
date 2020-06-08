package services

import (
	"encoding/json"
	"strings"

	"github.com/elastic/e2e-testing/cli/shell"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
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

// Run a kubectl command and return the output
func (k *Kubectl) Run(args ...string) (string, error) {
	return shell.Execute(".", "kubectl", args...)
}

// JsonToObj Convert a JSON string to a map[string]interface{}.
func (k *Kubectl) JsonToObj(jsonValue string) (map[string]interface{}, error) {
	var obj map[string]interface{}
	err := json.Unmarshal([]byte(jsonValue), &obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// YamlToObj Convert a YAML string to a map[string]interface{}.
func (k *Kubectl) YamlToObj(yamlValue string) (map[string]interface{}, error) {
	var obj map[string]interface{}
	err := yaml.Unmarshal([]byte(yamlValue), &obj)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"output": yamlValue,
		}).Error("Could not unmarshal YAML")

		return nil, err
	}

	return obj, nil
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

	jsonObj, err := k.JsonToObj(output)
	if err != nil {
		return "", err
	}

	status := jsonObj["status"].(map[string]interface{})
	return status["targetSelector"].(string), nil
}

// GetResourcesBySelector Use kubectl to get a resource identified by a selector, return the resource in a map[string]interface{}.
func (k *Kubectl) GetResourcesBySelector(resourceType, selector string) (map[string]interface{}, error) {
	output, err := k.Run("get", resourceType, "--selector", selector, "-o", "json")
	if err != nil {
		return nil, err
	}

	return k.JsonToObj(output)
}

// Describe Use kubecontrol describe command to get the description of a resource identified by a selector, return the resource in a map[string]interface{}.
func (k *Kubectl) Describe(resourceType, selector string) (map[string]interface{}, error) {
	output, err := k.Run("describe", resourceType, "--selector", selector)
	if err != nil {
		return nil, err
	}

	return k.YamlToObj(output)
}
