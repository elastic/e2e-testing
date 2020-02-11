package cmd

import (
	"errors"
	"path"
	"strings"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
	git "github.com/elastic/metricbeat-tests-poc/cli/internal"
	"github.com/spf13/cobra"
)

var remote = "elastic:master"

func init() {
	config.InitConfig()

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
	Run: func(cmd *cobra.Command, args []string) {
		workspace := config.Op.Workspace

		// BeatsRepo default object representing Beats project
		var BeatsRepo = git.ProjectBuilder.
			WithBaseWorkspace(path.Join(workspace, "git")).
			WithGitProtocol().
			WithDomain("github.com").
			WithName("beats").
			WithRemote(remote).
			Build()

		git.Clone(BeatsRepo)
	},
}
