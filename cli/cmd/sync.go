package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/elastic/metricbeat-tests-poc/cli/config"
	git "github.com/elastic/metricbeat-tests-poc/cli/internal"
	io "github.com/elastic/metricbeat-tests-poc/cli/internal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"gopkg.in/yaml.v2"
)

var deleteRepository = false
var excludedBlocks = []string{"build"}
var remote = "elastic:master"

func init() {
	config.InitConfig()

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

		if deleteRepository {
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

// tells whether the a array contains the x string.
func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// CopyComposeFiles copies only those services that has a supported-versions.yml
// file from Beats integrations, and we will need to copy them into a directory
// named as the original service (i.e. aerospike) under this tool's workspace,
// alongside the services. Besides that, the method will copy the _meta directory
// for each service, also sanitising the compose files: it will remove the 'build'
// blocks from the compose files.
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

		// discard error in any case
		_ = io.MkdirAll(filepath.Dir(targetFile))

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

		err = sanitizeComposeFile(targetFile)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	}
}

type service interface{}
type compose struct {
	Version  string             `yaml:"version"`
	Services map[string]service `yaml:"services"`
}

// removes non-needed blocks in the target compose file, such as the build context
func sanitizeComposeFile(composeFilePath string) error {
	bytes, err := io.ReadFile(composeFilePath)
	if err != nil {
		log.WithFields(log.Fields{
			"docker-compose": composeFilePath,
		}).Error("Could not read docker compose file")
		return err
	}

	c := compose{}
	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		log.WithFields(log.Fields{
			"docker-compose": composeFilePath,
		}).Error("Could not unmarshal docker compose file")
		return err
	}

	// we'll copy all fields but the excluded ones in this struct
	output := make(map[string]service)

	for k, srv := range c.Services {
		switch i := srv.(type) {
		case map[interface{}]interface{}:
			log.WithFields(log.Fields{
				"name":         k,
				"compose-file": composeFilePath,
			}).Debug("sanitize service in docker-compose file")

			for key, value := range i {
				strKey := fmt.Sprintf("%v", key)

				// remove the build context element
				if contains(excludedBlocks, strKey) {
					continue
				}

				output[strKey] = value
			}
		default:
			// skip
		}

		c.Services[k] = output
	}

	d, err := yaml.Marshal(&c)
	if err != nil {
		return err
	}
	io.WriteFile(d, composeFilePath)

	return nil
}
