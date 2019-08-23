package cmd

import (
	"github.com/elastic/metricbeat-tests-poc/cli/config"
	"github.com/elastic/metricbeat-tests-poc/cli/services"

	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var versionToRun string

func init() {
	config.InitConfig()

	rootCmd.AddCommand(runCmd)

	for k, srv := range config.AvailableServices() {
		serviceSubcommand := buildRunServiceCommand(k, srv)

		serviceSubcommand.Flags().StringVarP(&versionToRun, "version", "v", srv.Version, "Sets the image version to run")

		runServiceCmd.AddCommand(serviceSubcommand)
	}

	runCmd.AddCommand(runServiceCmd)

	for k, stack := range config.AvailableStacks() {
		stackSubcommand := buildRunStackCommand(k, stack)

		runStackCmd.AddCommand(stackSubcommand)
	}

	runCmd.AddCommand(runStackCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs a Service or Stack",
	Long:  "Runs a Service or Stack, spinning up Docker containers exposing its internal configuration so that you are able to connect to it in an easy manner",
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}

func buildRunServiceCommand(srv string, service config.Service) *cobra.Command {
	return &cobra.Command{
		Use:   srv,
		Short: `Runs a ` + srv + ` service`,
		Long:  `Runs a ` + srv + ` service, spinning up a Docker container for it and exposing its internal configuration so that you are able to connect to it in an easy manner`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := services.NewServiceManager()

			s := serviceManager.Build(srv, versionToRun, true)

			serviceManager.Run(s)
		},
	}
}

func buildRunStackCommand(key string, stack config.Stack) *cobra.Command {
	return &cobra.Command{
		Use:   key,
		Short: `Runs the ` + stack.Name + ` stack`,
		Long:  `Runs the ` + stack.Name + ` stack, spinning up the Services that compound it`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := services.NewServiceManager()

			availableServices := config.AvailableServices()
			if len(stack.Services) == 0 {
				log.WithFields(log.Fields{
					"command": "run",
					"stack":   key,
				}).Fatal("The Stack does not contain services. Please check configuration files")
			}

			servicesToRun := map[string]services.Service{}

			for k, srv := range stack.Services {
				originalSrv := availableServices[k]
				if !srv.Equals(originalSrv) {
					mergo.Merge(&srv, originalSrv)
				}

				srv.Name = srv.Name + "-" + key
				srv.Daemon = true
				srv.Labels = map[string]string{
					"stack": stack.Name,
				}
				s := serviceManager.BuildFromConfig(srv)

				if k == "elasticsearch" {
					serviceManager.Run(s)
				} else {
					servicesToRun[k] = s
				}
			}

			for _, srv := range servicesToRun {
				serviceManager.Run(srv)
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

var runStackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Allows to run a Stack, defined as subcommands",
	Long:  `Allows to run a Stack, defined as subcommands, and compounded by different services that cooperate between them`,
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}
