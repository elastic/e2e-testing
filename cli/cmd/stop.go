// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"

	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/internal/deploy"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

func init() {
	config.Init()

	rootCmd.AddCommand(stopCmd)

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

func buildStopProfileCommand(key string, profile config.Profile) *cobra.Command {
	return &cobra.Command{
		Use:   key,
		Short: `Stops the ` + profile.Name + ` profile`,
		Long:  `Stops the ` + profile.Name + ` profile, stopping the Services that compound it`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := deploy.NewServiceManager()

			err := serviceManager.StopCompose(
				context.Background(), true, []deploy.ServiceRequest{deploy.NewServiceRequest(key)})
			if err != nil {
				log.WithFields(log.Fields{
					"profile": key,
				}).Error("Could not stop the profile.")
			}
		},
	}
}

var stopProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Allows to stop a Profile, defined as subcommands",
	Long:  `Allows to stop a Profile, defined as subcommands, stopping all different services that compound the profile`,
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}
