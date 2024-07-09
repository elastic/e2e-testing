// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubectl

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/elastic/e2e-testing/internal/shell"
	"go.elastic.co/apm/v2"
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
func (k *Kubectl) GetStringResourcesBySelector(ctx context.Context, resourceType, selector string) (string, error) {
	span, _ := apm.StartSpanOptions(ctx, "Getting String resource by selector", "kubectl.stringResources.getBySelector", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	output, err := k.Run(ctx, "get", resourceType, "--selector="+selector, "-o", "json")
	if err != nil {
		return "", err
	}

	return output, nil
}

// GetResourcesBySelector Use kubectl to get a resource identified by a selector, return the resource in a map[string]interface{}.
func (k *Kubectl) GetResourcesBySelector(ctx context.Context, resourceType, selector string) (map[string]interface{}, error) {
	span, _ := apm.StartSpanOptions(ctx, "Getting resource by selector", "kubectl.resources.getBySelector", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	output, err := k.Run(ctx, "get", resourceType, "--selector="+selector, "-o", "json")
	if err != nil {
		return nil, err
	}

	return jsonToObj(output)
}

// GetResourceJSONPath use kubectl to get a resource by type and name, and a jsonpath, and return the definition of the resource in JSON.
func (k *Kubectl) GetResourceJSONPath(ctx context.Context, resourceType, resource, jsonPath string) (string, error) {
	span, _ := apm.StartSpanOptions(ctx, "Getting resource by JSON path", "kubectl.resources.getByJSONPath", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	output, err := k.Run(ctx, "get", resourceType, resource, "-o", "jsonpath='"+jsonPath+"'")
	if err != nil {
		return "{}", err
	}
	return output, nil
}

// GetResourceSelector use kubectl to get the selector for a resource identified by type and name.
func (k *Kubectl) GetResourceSelector(ctx context.Context, resourceType, resource string) (string, error) {
	span, _ := apm.StartSpanOptions(ctx, "Getting resource selector", "kubectl.resources.getSelector", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	output, err := k.GetResourceJSONPath(ctx, resourceType, resource, "{.metadata.labels.app}")
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(output, "'", ""), nil
}

// Run a kubectl command and return the output
func (k *Kubectl) Run(ctx context.Context, args ...string) (string, error) {
	return shell.Execute(ctx, ".", "kubectl", args...)
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
