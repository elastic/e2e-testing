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

var versionToStop string

func init() {
	config.Init()

	rootCmd.AddCommand(stopCmd)

	for k := range config.AvailableServices() {
		serviceSubcommand := buildStopServiceCommand(k)

		serviceSubcommand.Flags().StringVarP(&versionToStop, "version", "v", "latest", "Sets the image version to stop")

		stopServiceCmd.AddCommand(serviceSubcommand)
	}

	stopCmd.AddCommand(stopServiceCmd)

	for k, profile := range config.AvailableProfiles() {
		profileSubcommand := buildStopProfileCommand(k, profile)

		stopProfileCmd.AddCommand(profileSubcommand)
	}

	stopCmd.AddCommand(stopProfileCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops a Service or Profile",
	Long:  "Stops a Service or Profile, stoppping the Docker containers that expose their internal configuration",
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}

func buildStopServiceCommand(srv string) *cobra.Command {
	return &cobra.Command{
		Use:   srv,
		Short: `Stops a ` + srv + ` service`,
		Long:  `Stops a ` + srv + ` service, stoppping its Docker container`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := deploy.NewServiceManager()

			err := serviceManager.StopCompose(context.Background(), deploy.NewServiceRequest(srv))
			if err != nil {
				log.WithFields(log.Fields{
					"service": srv,
				}).Error("Could not stop the service.")
			}
		},
	}
}

func buildStopProfileCommand(key string, profile config.Profile) *cobra.Command {
	return &cobra.Command{
		Use:   key,
		Short: `Stops the ` + profile.Name + ` profile`,
		Long:  `Stops the ` + profile.Name + ` profile, stopping the Services that compound it`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := deploy.NewServiceManager()

			err := serviceManager.StopCompose(context.Background(), deploy.NewServiceRequest(key))
			if err != nil {
				log.WithFields(log.Fields{
					"profile": key,
				}).Error("Could not stop the profile.")
			}
		},
	}
}

var stopServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "Allows to stop a service, defined as subcommands",
	Long:  `Allows to stop a service, defined as subcommands, stopping the Docker containers for them.`,
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}

var stopProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Allows to stop a Profile, defined as subcommands",
	Long:  `Allows to stop a Profile, defined as subcommands, stopping all different services that compound the profile`,
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}
