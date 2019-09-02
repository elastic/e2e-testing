package cmd

import (
	"github.com/elastic/metricbeat-tests-poc/cli/config"
	"github.com/elastic/metricbeat-tests-poc/cli/services"

	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var versionToStop string

func init() {
	config.InitConfig()

	rootCmd.AddCommand(stopCmd)

	for k, srv := range config.AvailableServices() {
		serviceSubcommand := buildStopServiceCommand(k, srv)

		serviceSubcommand.Flags().StringVarP(&versionToStop, "version", "v", srv.Version, "Sets the image version to stop")

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

func buildStopServiceCommand(srv string, service config.Service) *cobra.Command {
	return &cobra.Command{
		Use:   srv,
		Short: `Stops a ` + srv + ` service`,
		Long:  `Stops a ` + srv + ` service, stoppping its Docker container`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := services.NewServiceManager()

			s := serviceManager.Build(srv, versionToStop, true)

			serviceManager.Stop(s)
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

			services := config.AvailableServices()
			if len(stack.Services) == 0 {
				log.WithFields(log.Fields{
					"command": "stop",
					"stack":   key,
				}).Fatal("The Stack does not contain services. Please check configuration files")
			}

			for k, srv := range stack.Services {
				originalSrv := services[k]
				if !srv.Equals(originalSrv) {
					mergo.Merge(&originalSrv, srv)
				}

				originalSrv.Name = originalSrv.Name + "-" + key
				originalSrv.Daemon = true
				s := serviceManager.BuildFromConfig(originalSrv)
				serviceManager.Stop(s)
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
