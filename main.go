package main

import (
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/elastic/metricbeat-tests-poc/cmd"
	"github.com/elastic/metricbeat-tests-poc/config"
	"github.com/elastic/metricbeat-tests-poc/docker"
	"github.com/elastic/metricbeat-tests-poc/services"
)

// checkWorkspace creates this tool workspace under user's home, in a hidden directory named ".op"
func checkWorkspace() {
	usr, _ := user.Current()

	w := filepath.Join(usr.HomeDir, ".op")

	if _, err := os.Stat(w); os.IsNotExist(err) {
		err = os.MkdirAll(w, 0755)
		if err != nil {
			log.Fatalf("Cannot create workdir for 'op' at "+w, err)
		}

		log.Println("'op' workdir created at " + w)
	}

	config.OpWorkspace = w

	serviceManager := services.NewServiceManager()

	config.NewConfig(w, serviceManager.AvailableServices())
}

func init() {
	checkWorkspace()

	docker.GetDevNetwork()
}

func main() {
	cmd.Execute()
}
