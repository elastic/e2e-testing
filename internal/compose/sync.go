package compose

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/internal/git"
	"github.com/elastic/e2e-testing/internal/io"
	"github.com/elastic/e2e-testing/internal/sanitizer"
	log "github.com/sirupsen/logrus"
	"path"
	"strings"

	"gopkg.in/yaml.v2"
)

// SyncMetricbeatComposeFiles first tries to fetch and checkout the specified Beats branch
// If that doesn't work, it clones the repo. Then it copies over all the metricbeat modules
// docker-compose files
func SyncMetricbeatComposeFiles(deleteRepo bool, remote string) error {
	workspace := config.Op.Workspace

	// beatsRepo default object representing Beats project
	beatsRepo := git.ProjectBuilder.
		WithBaseWorkspace(path.Join(workspace, "git")).
		WithDomain("github.com").
		WithName("beats").
		WithRemote(remote).
		Build()

	repoDir := path.Join(workspace, "git", beatsRepo.Name)

	if !deleteRepo {
		log.Trace("Fetching beats repo...")
		err := git.Fetch(beatsRepo)
		if err != nil {
			// this might happen if repo has been cloned previously with single branch option
			// this is made for backwards compatibility, in the future it will be possible
			// to just perform a os.Stat to know whether to fetch or clone
			log.WithFields(log.Fields{
				"branch":  beatsRepo.Branch,
				"details": "this might be expected, but you should not see this error every time",
			}).Warn(fmt.Sprintf("Git feth failed: %s", err.Error()))
			deleteRepo = true
		}
	}

	if deleteRepo {
		log.WithFields(log.Fields{
			"path": repoDir,
		}).Trace("Removing repository")
		os.RemoveAll(repoDir)

		log.Trace("Cloning beats repo...")
		err := git.Clone(beatsRepo)

		if err != nil {
			return err
		}
	}

	pattern := path.Join(
		beatsRepo.GetWorkspace(), "metricbeat", "module", "*", "_meta", "supported-versions.yml")
	copyIntegrationsComposeFiles(pattern, workspace)

	xPackPattern := path.Join(
		beatsRepo.GetWorkspace(), "x-pack", "metricbeat", "module", "*", "_meta", "supported-versions.yml")
	copyIntegrationsComposeFiles(xPackPattern, workspace)

	return nil
}

// copyIntegrationsComposeFiles copies only those services that has a supported-versions.yml
// file from Beats integrations, and we will need to copy them into a directory
// named as the original service (i.e. aerospike) under this tool's workspace,
// alongside the services. Besides that, the method will copy the _meta directory
// for each service, also sanitising the compose files: it will remove the 'build'
// blocks from the compose files.
func copyIntegrationsComposeFiles(pattern string, target string) {
	files := io.FindFiles(pattern)

	for _, file := range files {
		metaDir := filepath.Dir(file)
		serviceDir := filepath.Dir(metaDir)
		service := filepath.Base(serviceDir)

		composeFile := filepath.Join(serviceDir, "docker-compose.yml")
		configFile := filepath.Join(serviceDir, "_meta", "config.yml")
		targetFile := filepath.Join(
			target, "compose", "services", service, "docker-compose.yml")
		targetConfigFile := filepath.Join(
			target, "compose", "services", service, "_meta", "config.yml")

		err := io.MkdirAll(filepath.Dir(targetFile))
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  file,
			}).Warn("File was not copied")
			continue
		}

		err = io.CopyFile(composeFile, targetFile, 10000)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  composeFile,
			}).Warn("File was not copied")
			continue
		}

		err = io.CopyFile(configFile, targetConfigFile, 10000)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  configFile,
			}).Warn("File was not copied")
			continue
		}

		targetMetaDir := filepath.Join(target, "compose", "services", service, "_meta")
		err = io.CopyDir(metaDir, targetMetaDir)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"_meta": metaDir,
			}).Warn("Meta dir was not copied")
			continue
		}
		log.WithFields(log.Fields{
			"meta":       metaDir,
			"targetMeta": targetMetaDir,
		}).Trace("Integration _meta directory copied")

		err = sanitizeComposeFile(targetFile, targetFile)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  targetFile,
			}).Warn("Could not sanitize compose file")
			continue
		}
		log.WithFields(log.Fields{
			"file":       file,
			"targetFile": targetFile,
		}).Trace("Integration compose file copied")

		err = sanitizeConfigurationFile(service, targetConfigFile)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  targetFile,
			}).Warn("Could not sanitize config file")
			continue
		}
		log.WithFields(log.Fields{
			"file":       file,
			"targetFile": targetConfigFile,
		}).Trace("Integration config file copied")
	}

	log.WithFields(log.Fields{
		"files":  len(files),
		"target": target,
	}).Info("Integrations files copied")
}

var excludedBlocks = []string{"build"}

// removes non-needed blocks in the target compose file, such as the build context
func sanitizeComposeFile(composeFilePath string, targetFilePath string) error {
	bytes, err := io.ReadFile(composeFilePath)
	if err != nil {
		log.WithFields(log.Fields{
			"docker-compose": composeFilePath,
		}).Error("Could not read docker compose file")
		return err
	}

	c := composeFile{}
	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		log.WithFields(log.Fields{
			"docker-compose": composeFilePath,
		}).Error("Could not unmarshal docker compose file")
		return err
	}

	// sets version to 2.4 for testing purpose
	c.Version = "2.4"

	for k, srv := range c.Services {
		// we'll copy all fields but the excluded ones in this struct
		output := make(map[string]service)

		switch i := srv.(type) {
		case map[interface{}]interface{}:
			log.WithFields(log.Fields{
				"name":         k,
				"compose-file": composeFilePath,
			}).Trace("sanitize service in docker-compose file")

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

	return io.WriteFile(d, targetFilePath)
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

type service interface{}
type composeFile struct {
	Version  string             `yaml:"version"`
	Services map[string]service `yaml:"services"`
}

// sanitizeConfigurationFile replaces 127.0.0.1 with current service name
func sanitizeConfigurationFile(serviceName string, configFilePath string) error {
	bytes, err := io.ReadFile(configFilePath)
	if err != nil {
		log.WithFields(log.Fields{
			"config.yml": configFilePath,
		}).Error("Could not read config.yml file")
		return err
	}

	// replace local IP with service name, because it will be discovered under Docker Compose network
	content := strings.ReplaceAll(string(bytes), "127.0.0.1", serviceName)
	content = strings.ReplaceAll(content, "localhost", serviceName)
	// prepend modules header
	content = "metricbeat.modules:\n" + content

	serviceSanitizer := sanitizer.GetConfigSanitizer(serviceName)
	content = serviceSanitizer.Sanitize(content)

	log.WithFields(log.Fields{
		"config.reference.yml": configFilePath,
	}).Trace("Config file sanitized")

	return io.WriteFile([]byte(content), configFilePath)
}
