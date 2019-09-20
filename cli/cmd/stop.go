package cmd

import (
	"github.com/elastic/metricbeat-tests-poc/cli/config"
	"github.com/elastic/metricbeat-tests-poc/cli/services"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var versionToStop string

func init() {
	config.InitConfig()

	rootCmd.AddCommand(stopCmd)

	for k := range config.AvailableServices() {
		serviceSubcommand := buildStopServiceCommand(k)

		serviceSubcommand.Flags().StringVarP(&versionToStop, "version", "v", "latest", "Sets the image version to stop")

		stopServiceCmd.AddCommand(serviceSubcommand)
	}

	stopCmd.AddCommand(stopServiceCmd)

	for k, stack := range config.AvailableStacks() {
		stackSubcommand := buildStopStackCommand(k, stack)

		stopStackCmd.AddCommand(stackSubcommand)
	}

	stopCmd.AddCommand(stopStackCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops a Service or Stack",
	Long:  "Stops a Service or Stack, stoppping the Docker containers that expose their internal configuration",
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
			serviceManager := services.NewServiceManager()

			err := serviceManager.StopCompose(false, []string{srv})
			if err != nil {
				log.WithFields(log.Fields{
					"service": srv,
				}).Error("Could not stop the service.")
			}
		},
	}
}

func buildStopStackCommand(key string, stack config.Stack) *cobra.Command {
	return &cobra.Command{
		Use:   key,
		Short: `Stops the ` + stack.Name + ` stack`,
		Long:  `Stops the ` + stack.Name + ` stack, stopping the Services that compound it`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := services.NewServiceManager()

			err := serviceManager.StopCompose(true, []string{key})
			if err != nil {
				log.WithFields(log.Fields{
					"stack": key,
				}).Error("Could not stop the stack.")
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

var stopStackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Allows to stop a Stack, defined as subcommands",
	Long:  `Allows to stop a Stack, defined as subcommands, stopping all different services that compound the stack`,
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}
