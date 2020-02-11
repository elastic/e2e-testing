package internal

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// CopyComposeFiles copies files to a target directory. The files
// will representt the docker-compose.yml from Beats integrations,
// and we will need to copy them into a directory named as the original
// service (i.e. aerospike) under this tool's workspace, alongside
// the services
func CopyComposeFiles(files []string, target string) {
	for _, file := range files {
		serviceDir := filepath.Dir(file)
		service := filepath.Base(serviceDir)

		targetFile := filepath.Join(
			target, "compose", "services", service, "docker-compose.yml")

		err := copy(file, targetFile, 10000)
		if err != nil {
			log.WithFields(log.Fields{
				"error":  err,
				"file":   file,
				"target": target,
			}).Warn("File was not copied")
		}
	}
}

// FindFiles finds files recursively using a Glob pattern for the matching
func FindFiles(pattern string) []string {
	matches, err := filepath.Glob(pattern)

	if err != nil {
		log.WithFields(log.Fields{
			"pattern": pattern,
		}).Warn("pattern is not a Glob")

		return []string{}
	}

	return matches
}

// Optimising the copy of files in Go:
// https://opensource.com/article/18/6/copying-files-go
func copy(src string, dst string, BUFFERSIZE int64) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return errors.New(src + " is not a regular file")
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	_, err = os.Stat(dst)
	if err == nil {
		return errors.New("File " + dst + " already exists")
	}

	// check if parent dir for the file exist, otherwise create it
	parent := filepath.Dir(dst)
	if _, err := os.Stat(parent); os.IsNotExist(err) {
		err = os.MkdirAll(parent, 0755)
		if err != nil {
			return errors.New("File " + parent + " cannot be created")
		}
	}

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	buf := make([]byte, BUFFERSIZE)
	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}

	return err
}
