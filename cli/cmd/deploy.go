// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"

	"github.com/elastic/e2e-testing/internal/config"
	"github.com/elastic/e2e-testing/internal/deploy"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var deployToProfile string

func init() {
	config.Init()

	for k := range config.AvailableServices() {
		// deploy command
		deployServiceSubcommand := buildDeployServiceCommand(k)

		deployServiceSubcommand.Flags().StringVarP(&deployToProfile, "profile", "s", "", "Sets the profile where to deploy the service. (Required)")
		deployServiceSubcommand.Flags().StringVarP(&versionToRun, "version", "v", "latest", "Sets the image version to run")

		deployCmd.AddCommand(deployServiceSubcommand)

		// undeploy command
		undeployServiceSubcommand := buildUndeployServiceCommand(k)
		undeployServiceSubcommand.Flags().StringVarP(&deployToProfile, "profile", "s", "", "Sets the profile where to undeploy the service. (Required)")

		undeployCmd.AddCommand(undeployServiceSubcommand)
	}

	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(undeployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploys a Service to a Profile",
	Long:  "Deploys a Service to a Profile",
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}

var undeployCmd = &cobra.Command{
	Use:   "undeploy",
	Short: "Undeploys a Service from a Profile",
	Long:  "Undeploys a Service from a Profile",
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}

func buildDeployServiceCommand(srv string) *cobra.Command {
	return &cobra.Command{
		Use:   srv,
		Short: `Deploys a ` + srv + ` service`,
		Long:  `Deploys a ` + srv + ` service, adding it to a running profile, identified by its name`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := deploy.NewServiceManager()

			env := map[string]string{}
			env = config.PutServiceEnvironment(env, srv, versionToRun)

			err := serviceManager.AddServicesToCompose(
				context.Background(),
				deploy.NewServiceRequest(deployToProfile),
				[]deploy.ServiceRequest{deploy.NewServiceRequest(srv)},
				env)
			if err != nil {
				log.WithFields(log.Fields{
					"profile":  deployToProfile,
					"services": servicesToRun,
				}).Error("Could not add services to the profile.")
			}
		},
	}
}

func buildUndeployServiceCommand(srv string) *cobra.Command {
	return &cobra.Command{
		Use:   srv,
		Short: `Undeploys a ` + srv + ` service`,
		Long:  `Undeploys a ` + srv + ` service, removing it from a running profile, identified by its name`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := deploy.NewServiceManager()

			env := map[string]string{}
			env = config.PutServiceEnvironment(env, srv, versionToRun)

			err := serviceManager.RemoveServicesFromCompose(
				context.Background(),
				deploy.NewServiceRequest(deployToProfile),
				[]deploy.ServiceRequest{deploy.NewServiceRequest(srv)},
				env)
			if err != nil {
				log.WithFields(log.Fields{
					"profile":  deployToProfile,
					"services": servicesToRun,
				}).Error("Could not remove services from the profile.")
			}
		},
	}
}
