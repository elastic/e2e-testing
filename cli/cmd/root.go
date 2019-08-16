package cmd

import (
	"github.com/spf13/cobra"

	"github.com/elastic/metricbeat-tests-poc/cli/log"
	"github.com/elastic/metricbeat-tests-poc/cli/services"
)

var serviceManager = services.NewServiceManager()

var rootCmd = &cobra.Command{
	Use:   "op",
	Short: "op (Observability Provisioner) makes it easier to develop Observability projects.",
	Long: `A Fast and Flexible CLI for developing and testing Elastic's Observability projects
	built with ❤️ by mdelapenya and friends in Go.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

// Execute execute root command
func Execute() {
	err := rootCmd.Execute()
	log.CheckIfError(err)
}
