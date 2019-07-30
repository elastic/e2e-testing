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

	subcommands := []*cobra.Command{
		runApacheCmd, runkafkaCmd, runMysqlCmd, runStackCmd,
	}

	for i := 0; i < len(subcommands); i++ {
		subcommand := subcommands[i]

		runCmd.AddCommand(subcommand)

		subcommand.Flags().StringVarP(&versionToRun, "version", "v", "", "Sets the image version to run")
	}

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

var runApacheCmd = &cobra.Command{
	Use:   "apache",
	Short: "Runs an Apache service",
	Long: `Runs an Apache service to be monitored by Metricbeat, spinning up a Docker container for it and exposing its internal
	configuration so that you are able to connect to it in an easy manner`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("run requires zero or one argument representing the image tag to be run")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		s := services.NewApacheService(versionToRun, true)

		serviceManager := services.NewServiceManager()

		serviceManager.Run(s)
	},
}

var runkafkaCmd = &cobra.Command{
	Use:   "kafka",
	Short: "Runs a Kafka service",
	Long: `Runs a Kafka service to be monitored by Metricbeat, spinning up a Docker container for it and exposing its internal
	configuration so that you are able to connect to it in an easy manner`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("run requires zero or one argument representing the image tag to be run")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		s := services.NewKafkaService(versionToRun, true)

		serviceManager := services.NewServiceManager()

		serviceManager.Run(s)
	},
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
		serviceManager := services.NewServiceManager()

		es := services.NewElasticsearchService(versionToRun, true)
		serviceManager.Run(es)

		s := services.NewKibanaService(versionToRun, true, es)
		serviceManager.Run(s)
	},
}

var runMysqlCmd = &cobra.Command{
	Use:   "mysql",
	Short: "Runs a MySQL service",
	Long: `Runs a MySQL service to be monitored by Metricbeat, spinning up a Docker container for it and exposing its internal
	configuration so that you are able to connect to it in an easy manner`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("run requires zero or one argument representing the image tag to be run")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		s := services.NewMySQLService(versionToRun, true)

		serviceManager := services.NewServiceManager()

		serviceManager.Run(s)
	},
}
