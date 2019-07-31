package cmd

import (
	"errors"
	"fmt"

	"github.com/elastic/metricbeat-tests-poc/services"

	"github.com/spf13/cobra"
)

var versionToStop string

func init() {
	rootCmd.AddCommand(stopCmd)

	initialServices := []string{
		"apache", "kafka", "metricbeat", "mysql",
	}

	for _, s := range initialServices {
		stopSubcommand := buildStopServiceCommand(s)

		stopSubcommand.Flags().StringVarP(&versionToStop, "version", "v", "", "Sets the image version to stop")

		stopCmd.AddCommand(stopSubcommand)
	}

	stopCmd.AddCommand(stopStackCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops a Service to be monitored",
	Long: `Stops a Service monitored by Metricbeat, stoppping the Docker container for it that exposes its internal
	configuration so that you are able to connect to it in an easy manner`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("run requires zero or one argument representing the image tag to be run")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello stop!")
	},
}

func buildStopServiceCommand(service string) *cobra.Command {
	return &cobra.Command{
		Use:   service,
		Short: `Stops a ` + service + ` service`,
		Long: `Stops a ` + service + ` service to be monitored by Metricbeat, stoppping the Docker container for it that exposes its internal
		configuration so that you are able to connect to it in an easy manner`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("run requires zero or one argument representing the image tag to be run")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			serviceManager := services.NewServiceManager()

			s := serviceManager.Build(service, versionToStop)

			serviceManager.Stop(s)
		},
	}
}

var stopStackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Stops an Elastic Stack (Elasticsearch + Kibana)",
	Long: `Stops an Elastic Stack (Elasticsearch + Kibana), stoppping the Docker containers for it that exposes its internal
	configuration so that you are able to connect to it in an easy manner`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("run requires zero or one argument representing the image tag to be run")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		serviceManager := services.NewServiceManager()

		es := serviceManager.Build("elasticsearch", versionToStop)
		kibana := serviceManager.Build("kibana", versionToStop)

		serviceManager.Stop(kibana)
		serviceManager.Stop(es)
	},
}
