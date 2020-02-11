package cmd

import (
	"errors"
	"os"
	"path"
	"strings"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
	git "github.com/elastic/metricbeat-tests-poc/cli/internal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var remote = "elastic:master"
var remove = false

func init() {
	config.InitConfig()

	syncIntegrationsCmd.Flags().StringVarP(&remote, "remote", "r", "elastic:master", "Sets the remote for Beats, using 'user:branch' as format (i.e. elastic:master)")
	syncIntegrationsCmd.Flags().BoolVarP(&remove, "remove", "R", false, "Will remove the existing Beats repository before cloning it again")

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

		if remove {
			repoDir := path.Join(workspace, "git", BeatsRepo.Name)

			log.WithFields(log.Fields{
				"path": repoDir,
			}).Debug("Removing repository")
			os.RemoveAll(repoDir)
			log.WithFields(log.Fields{
				"path": repoDir,
			}).Debug("Repository removed")
		}

		git.Clone(BeatsRepo)
	},
}
