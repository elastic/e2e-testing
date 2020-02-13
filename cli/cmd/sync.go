package cmd

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
	git "github.com/elastic/metricbeat-tests-poc/cli/internal"
	io "github.com/elastic/metricbeat-tests-poc/cli/internal"
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

		copyIntegrationsComposeFiles(BeatsRepo, workspace)
	},
}

// CopyComposeFiles copies only those services that has a supported-versions.yml
// file from Beats integrations, and we will need to copy them into a directory
// named as the original service (i.e. aerospike) under this tool's workspace,
// alongside the services. Besides that, the method will copy the _meta directory
// for each service
func copyIntegrationsComposeFiles(beats git.Project, target string) {
	pattern := path.Join(
		beats.GetWorkspace(), "metricbeat", "module", "*", "_meta", "supported-versions.yml")

	files := io.FindFiles(pattern)

	for _, file := range files {
		metaDir := filepath.Dir(file)
		serviceDir := filepath.Dir(metaDir)
		service := filepath.Base(serviceDir)

		composeFile := filepath.Join(serviceDir, "docker-compose.yml")
		targetFile := filepath.Join(
			target, "compose", "services", service, "docker-compose.yml")

		err := io.CopyFile(composeFile, targetFile, 10000)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  file,
			}).Warn("File was not copied")
		}

		targetMetaDir := filepath.Join(target, "compose", "services", service, "_meta")
		err = io.CopyDir(metaDir, targetMetaDir)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"_meta": metaDir,
			}).Warn("Meta dir was not copied")
		}
	}
}
