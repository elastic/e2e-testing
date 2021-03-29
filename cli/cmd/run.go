// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/cli/services"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var servicesToRun string
var versionToRun string

func init() {
	config.Init()

	rootCmd.AddCommand(runCmd)

	for k := range config.AvailableServices() {
		serviceSubcommand := buildRunServiceCommand(k)

		serviceSubcommand.Flags().StringVarP(&versionToRun, "version", "v", "latest", "Sets the image version to run")

		runServiceCmd.AddCommand(serviceSubcommand)
	}

	runCmd.AddCommand(runServiceCmd)

	for k, profile := range config.AvailableProfiles() {
		profileSubcommand := buildRunProfileCommand(k, profile)

		profileSubcommand.Flags().StringVarP(&versionToRun, "profileVersion", "v", "latest", "Sets the profile version to run")
		profileSubcommand.Flags().StringVarP(&servicesToRun, "withServices", "s", "", "Sets a list of comma-separated services to be depoyed alongside the profile")

		runProfileCmd.AddCommand(profileSubcommand)
	}

	runCmd.AddCommand(runProfileCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs a Service or Profile",
	Long:  "Runs a Service or Profile, spinning up Docker containers exposing its internal configuration so that you are able to connect to it in an easy manner",
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}

func buildRunServiceCommand(srv string) *cobra.Command {
	return &cobra.Command{
		Use:   srv,
		Short: `Runs a ` + srv + ` service`,
		Long:  `Runs a ` + srv + ` service, spinning up a Docker container for it and exposing its internal configuration so that you are able to connect to it in an easy manner`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := services.NewServiceManager()

			env := config.PutServiceEnvironment(map[string]string{}, srv, versionToRun)

			err := serviceManager.RunCompose(context.Background(), false, []string{srv}, env)
			if err != nil {
				log.WithFields(log.Fields{
					"service": srv,
				}).Error("Could not run the service.")
			}
		},
	}
}

func buildRunProfileCommand(key string, profile config.Profile) *cobra.Command {
	return &cobra.Command{
		Use:   key,
		Short: `Runs the ` + profile.Name + ` profile`,
		Long: `Runs the ` + profile.Name + ` profile, spinning up the Services that compound it

Example:
  go run main.go run profile fleet -s elastic-agent:8.0.0-SNAPSHOT
`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := services.NewServiceManager()

			env := map[string]string{
				"profileVersion": versionToRun,
			}

			err := serviceManager.RunCompose(context.Background(), true, []string{key}, env)
			if err != nil {
				log.WithFields(log.Fields{
					"profile": key,
				}).Error("Could not run the profile.")
			}

			composeNames := []string{}
			if servicesToRun != "" {
				services := strings.Split(servicesToRun, ",")

				for _, srv := range services {
					arr := strings.Split(srv, ":")
					if len(arr) != 2 {
						log.WithFields(log.Fields{
							"profile":  key,
							"services": servicesToRun,
						}).Error("Unable to determine the <image>:<tag>, please make sure to use a known docker tag format, eg. `elastic-agent:8.0.0-SNAPSHOT`")
						os.Exit(1)
					}
					image := arr[0]
					tag := arr[1]

					env = config.PutServiceEnvironment(env, image, tag)
					composeNames = append(composeNames, image)
				}

				err = serviceManager.AddServicesToCompose(context.Background(), key, composeNames, env)
				if err != nil {
					log.WithFields(log.Fields{
						"profile":  key,
						"services": servicesToRun,
					}).Error("Could not add services to the profile.")
				}
			}
		},
	}
}

var runServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "Allows to run a service, defined as subcommands",
	Long:  `Allows to run a service, defined as subcommands, spinning up Docker containers for them and exposing their internal configuration so that you are able to connect to them in an easy manner`,
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}

var runProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Allows to run a Profile, defined as subcommands",
	Long:  `Allows to run a Profile, defined as subcommands, and compounded by different services that cooperate between them`,
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}
