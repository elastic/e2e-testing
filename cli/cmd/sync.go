package cmd

import (
	"path"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
	git "github.com/elastic/metricbeat-tests-poc/cli/internal"
	"github.com/spf13/cobra"
)

func init() {
	config.InitConfig()

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
	Run: func(cmd *cobra.Command, args []string) {
		workspace := config.Op.Workspace

		// BeatsBuilder default object representing Beats project
		var BeatsBuilder = git.ProjectBuilder.
			WithBaseWorkspace(path.Join(workspace, "git")).
			WithGitProtocol().
			WithDomain("github.com").
			WithName("beats").
			WithCoords("elastic:master")

		git.Clone(BeatsBuilder.Build())
	},
}
