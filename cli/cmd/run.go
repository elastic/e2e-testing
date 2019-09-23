package cmd

import (
	"strings"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
	"github.com/elastic/metricbeat-tests-poc/cli/services"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var servicesToRun string
var versionToRun string

func init() {
	config.InitConfig()

	rootCmd.AddCommand(runCmd)

	for k := range config.AvailableServices() {
		serviceSubcommand := buildRunServiceCommand(k)

		serviceSubcommand.Flags().StringVarP(&versionToRun, "version", "v", "latest", "Sets the image version to run")

		runServiceCmd.AddCommand(serviceSubcommand)
	}

	runCmd.AddCommand(runServiceCmd)

	for k, stack := range config.AvailableStacks() {
		stackSubcommand := buildRunStackCommand(k, stack)

		stackSubcommand.Flags().StringVarP(&servicesToRun, "withServices", "s", "", "Sets a list of comma-separated services to be depoyed alongside the stack")

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

func buildRunServiceCommand(srv string) *cobra.Command {
	return &cobra.Command{
		Use:   srv,
		Short: `Runs a ` + srv + ` service`,
		Long:  `Runs a ` + srv + ` service, spinning up a Docker container for it and exposing its internal configuration so that you are able to connect to it in an easy manner`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := services.NewServiceManager()

			env := map[string]string{
				srv + "Tag": versionToRun,
			}

			err := serviceManager.RunCompose(false, []string{srv}, env)
			if err != nil {
				log.WithFields(log.Fields{
					"service": srv,
				}).Error("Could not run the service.")
			}
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

			err := serviceManager.RunCompose(true, []string{key}, map[string]string{})
			if err != nil {
				log.WithFields(log.Fields{
					"stack": key,
				}).Error("Could not run the stack.")
			}

			composeNames := []string{}
			env := map[string]string{}
			if servicesToRun != "" {
				services := strings.Split(servicesToRun, ",")

				for _, srv := range services {
					arr := strings.Split(srv, ":")
					image := arr[0]
					tag := arr[1]

					env[image+"Tag"] = tag
					composeNames = append(composeNames, image)
				}

				err = serviceManager.AddServicesToCompose(key, composeNames, env)
				if err != nil {
					log.WithFields(log.Fields{
						"stack":    key,
						"services": servicesToRun,
					}).Error("Could not add services to the stack.")
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

var runStackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Allows to run a Stack, defined as subcommands",
	Long:  `Allows to run a Stack, defined as subcommands, and compounded by different services that cooperate between them`,
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}
