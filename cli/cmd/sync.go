// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"errors"
	"github.com/elastic/e2e-testing/internal/compose"
	"strings"

	"github.com/elastic/e2e-testing/cli/config"
	"github.com/spf13/cobra"
)

var deleteRepository = false
var remote = "elastic:master"

func init() {
	config.Init()

	syncIntegrationsCmd.Flags().BoolVarP(&deleteRepository, "delete", "d", false, "Will delete the existing Beats repository before cloning it again (default false)")
	syncIntegrationsCmd.Flags().StringVarP(&remote, "remote", "r", "elastic:master", "Sets the remote for Beats, using 'user:branch' as format (i.e. elastic:master)")

	syncCmd.AddCommand(syncIntegrationsCmd)
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync services from Beats",
	Long:  "Subcommands will allow synchronising services",
	Run: func(cmd *cobra.Command, args []string) {
		// NOOP
	},
}

var syncIntegrationsCmd = &cobra.Command{
	Use:   "integrations",
	Short: "Sync services from Beats",
	Long:  "Sync services from Beats, checking out current version of the services from GitHub",
	Args: func(cmd *cobra.Command, args []string) error {
		arr := strings.Split(remote, ":")
		if len(arr) == 2 {
			return nil
		}
		return errors.New("invalid 'user:branch' format: " + remote + ". Example: 'elastic:master'")
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return compose.SyncMetricbeatComposeFiles(deleteRepository, remote)
	},
}
