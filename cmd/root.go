package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mt",
	Short: "mt (Metricbeat tests) makes it easier to test Metricbeat's modules.",
	Long: `A Fast and Flexible CLI for testing Metricbeat's modules
				built with love by mdelapenya and friends in Go.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

// Execute execute root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
