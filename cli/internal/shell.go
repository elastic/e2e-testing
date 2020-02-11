package internal

import (
	"bytes"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Execute execute
func Execute(workspace string, command string, args ...string) string {
	cmd := exec.Command(command, args[0:]...)

	cmd.Dir = workspace

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.WithFields(log.Fields{
			"baseDir": workspace,
			"command": command,
			"args":    args,
		}).Fatal("Error executing command")
	}

	return strings.Trim(out.String(), "\n")
}
