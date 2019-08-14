package cmd

import (
	"errors"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
	"github.com/elastic/metricbeat-tests-poc/cli/services"

	"github.com/spf13/cobra"
)

var versionToRun string

func init() {
	config.InitConfig()

	rootCmd.AddCommand(runCmd)

	for k, srv := range config.AvailableServices() {
		serviceSubcommand := buildRunServiceCommand(k)

		serviceSubcommand.Flags().StringVarP(&versionToRun, "version", "v", srv.Version, "Sets the image version to run")

		runServiceCmd.AddCommand(serviceSubcommand)
	}

	runCmd.AddCommand(runServiceCmd)
	runCmd.AddCommand(runStackCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs a Service",
	Long: `Runs a Service, spinning up a Docker container for it and exposing its internal.
	configuration so that you are able to connect to it in an easy manner`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("run requires zero or one argument representing the image tag to be run")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}

func buildRunServiceCommand(service string) *cobra.Command {
	return &cobra.Command{
		Use:   service,
		Short: `Runs a ` + service + ` service`,
		Long: `Runs a ` + service + ` service, spinning up a Docker container for it and exposing its internal
		configuration so that you are able to connect to it in an easy manner`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("run requires zero or one argument representing the image tag to be run")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := services.NewServiceManager()

			s := serviceManager.Build(service, versionToRun, true)

			serviceManager.Run(s)
		},
	}
}

var runServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "Allows to run a service, defined as subcommands",
	Long: `Allows to run a service, defined as subcommands, spinning up Docker containers for them and exposing their internal
	configuration so that you are able to connect to them in an easy manner`,
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}

var runStackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Runs a Stack",
	Long: `Runs a Stack, compounded by different services that cooperate, spinning up Docker containers for them and exposing
	their internal configuration so that you are able to connect to them in an easy manner`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("run requires zero or one argument representing the image tag to be run")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}
