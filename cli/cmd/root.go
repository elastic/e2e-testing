// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

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
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error executing command")
	}
}
