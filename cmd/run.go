package cmd

import (
	"errors"
	"fmt"

	"github.com/elastic/metricbeat-tests-poc/services"

	"github.com/spf13/cobra"
)

var versionToRun string

func init() {
	rootCmd.AddCommand(runCmd)

	for k := range serviceManager.AvailableServices() {
		runSubcommand := buildRunServiceCommand(k)

		runSubcommand.Flags().StringVarP(&versionToRun, "version", "v", "", "Sets the image version to run")

		runCmd.AddCommand(runSubcommand)
	}

	runStackCmd.Flags().StringVarP(&versionToRun, "version", "v", "", "Sets the image version to run")

	runCmd.AddCommand(runStackCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs a Service to be monitored",
	Long: `Runs a Service to be monitored by Metricbeat, spinning up a Docker container for it and exposing its internal.
	configuration so that you are able to connect to it in an easy manner`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("run requires zero or one argument representing the image tag to be run")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello run!")
	},
}

func buildRunServiceCommand(service string) *cobra.Command {
	return &cobra.Command{
		Use:   service,
		Short: `Runs a ` + service + ` service`,
		Long: `Runs a ` + service + ` service to be monitored by Metricbeat, spinning up a Docker container for it and exposing its internal
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

var runStackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Runs an Elastic Stack (Elasticsearch + Kibana)",
	Long: `Runs an Elastic Stack (Elasticsearch + Kibana), spinning up Docker containers for them and exposing their internal
	configuration so that you are able to connect to them in an easy manner`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("run requires zero or one argument representing the image tag to be run")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		es := serviceManager.Build("elasticsearch", versionToRun, true)
		serviceManager.Run(es)

		s := services.RunKibanaService(versionToRun, true, es)
		serviceManager.Run(s)
	},
}
