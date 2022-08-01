// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

func (fts *FleetTestSuite) agentRunPolicy(policyName string) error {
	agentRunPolicyFn := func() error {
		agentService := deploy.NewServiceRequest(common.ElasticAgentServiceName)
		manifest, _ := fts.getDeployer().GetServiceManifest(fts.currentContext, agentService)

		policies, err := fts.kibanaClient.ListPolicies(fts.currentContext)
		if err != nil {
			return err
		}

		var policy *kibana.Policy
		for _, p := range policies {
			if policyName == p.Name {
				policy = &p
				break
			}
		}

		if policy == nil {
			return fmt.Errorf("policy not found '%s'", policyName)
		}

		agent, err := fts.kibanaClient.GetAgentByHostnameFromList(fts.currentContext, manifest.Hostname)
		if err != nil {
			return err
		}

		if agent.PolicyID != policy.ID {
			log.Errorf("FOUND %s %s", agent.PolicyID, policy.ID)
			return fmt.Errorf("agent not running the correct policy (running '%s' instead of '%s')", agent.PolicyID, policy.ID)
		}

		return nil
	}
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	exp := utils.GetExponentialBackOff(maxTimeout)

	return backoff.Retry(agentRunPolicyFn, exp)
}

func (fts *FleetTestSuite) agentUsesPolicy(policyName string) error {
	agentUsesPolicyFn := func() error {
		policies, err := fts.kibanaClient.ListPolicies(fts.currentContext)
		if err != nil {
			return err
		}

		for _, p := range policies {
			if policyName == p.Name {

				fts.Policy = p
				break
			}
		}

		if fts.Policy.Name != policyName {
			return fmt.Errorf("policy not found '%s'", policyName)
		}

		return nil
	}
	maxTimeout := time.Duration(utils.TimeoutFactor) * time.Minute * 2
	exp := utils.GetExponentialBackOff(maxTimeout)

	return backoff.Retry(agentUsesPolicyFn, exp)
}

// kibanaUsesProfile this step should be ideally called as a Background or a Given clause, so that it
// is executed before any other in the test scenario. It will configure the Kibana profile to be used
// in the scenario, changing the configuration file to be used.
func (fts *FleetTestSuite) kibanaUsesProfile(profile string) error {
	fts.KibanaProfile = profile

	env := fts.getProfileEnv()

	return bootstrapFleet(context.Background(), env)
}
