package cmd

import (
	"github.com/elastic/metricbeat-tests-poc/cli/config"
	"github.com/elastic/metricbeat-tests-poc/cli/services"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var deployToStack string

func init() {
	config.InitConfig()

	for k := range config.AvailableServices() {
		// deploy command
		deployServiceSubcommand := buildDeployServiceCommand(k)

		deployServiceSubcommand.Flags().StringVarP(&deployToStack, "stack", "s", "", "Sets the stack where to deploy the service. (Required)")
		deployServiceSubcommand.Flags().StringVarP(&versionToRun, "version", "v", "latest", "Sets the image version to run")

		deployCmd.AddCommand(deployServiceSubcommand)

		// undeploy command
		undeployServiceSubcommand := buildUndeployServiceCommand(k)
		undeployServiceSubcommand.Flags().StringVarP(&deployToStack, "stack", "s", "", "Sets the stack where to undeploy the service. (Required)")

		undeployCmd.AddCommand(undeployServiceSubcommand)
	}

	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(undeployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploys a Service to a Stack",
	Long:  "Deploys a Service to a Stack",
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}

var undeployCmd = &cobra.Command{
	Use:   "undeploy",
	Short: "Undeploys a Service from a Stack",
	Long:  "Undeploys a Service from a Stack",
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}

func buildDeployServiceCommand(srv string) *cobra.Command {
	return &cobra.Command{
		Use:   srv,
		Short: `Deploys a ` + srv + ` service`,
		Long:  `Deploys a ` + srv + ` service, adding it to a running stack, identified by its name`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := services.NewServiceManager()

			env := map[string]string{
				srv + "Tag": versionToRun,
			}

			err := serviceManager.AddServicesToCompose(deployToStack, []string{srv}, env)
			if err != nil {
				log.WithFields(log.Fields{
					"stack":    deployToStack,
					"services": servicesToRun,
				}).Error("Could not add services to the stack.")
			}
		},
	}
}

func buildUndeployServiceCommand(srv string) *cobra.Command {
	return &cobra.Command{
		Use:   srv,
		Short: `Undeploys a ` + srv + ` service`,
		Long:  `Undeploys a ` + srv + ` service, removing it from a running stack, identified by its name`,
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := services.NewServiceManager()

			err := serviceManager.RemoveServicesFromCompose(deployToStack, []string{srv})
			if err != nil {
				log.WithFields(log.Fields{
					"stack":    deployToStack,
					"services": servicesToRun,
				}).Error("Could not remove services from the stack.")
			}
		},
	}
}
