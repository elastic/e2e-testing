package internal

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// CopyComposeFiles copies compose files to a target directory. The files
// will represent the docker-compose.yml from Beats integrations, and we
// will need to copy them into a directory named as the original service
// (i.e. aerospike) under this tool's workspace, alongside the services.
// Besides that, the method will copy the _meta directory for each service
func CopyComposeFiles(files []string, target string) {
	for _, file := range files {
		serviceDir := filepath.Dir(file)
		service := filepath.Base(serviceDir)

		targetFile := filepath.Join(
			target, "compose", "services", service, "docker-compose.yml")

		err := copyFile(file, targetFile, 10000)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  file,
			}).Warn("File was not copied")
		}

		metaDir := filepath.Join(serviceDir, "_meta")
		targetMetaDir := filepath.Join(target, "compose", "services", service, "_meta")
		err = copyDir(metaDir, targetMetaDir)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"_meta": metaDir,
			}).Warn("Meta dir was not copied")
		}
	}
}

//MkdirAll creates all directories in a path
func MkdirAll(file string) error {
	// check if parent dir for the file exist, otherwise create it
	parent := filepath.Dir(file)
	if _, err := os.Stat(parent); os.IsNotExist(err) {
		err = os.MkdirAll(parent, 0755)
		if err != nil {
			return errors.New("File " + parent + " cannot be created")
		}
	}

	return nil
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

// copyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory must *not* exist.
// Symlinks are ignored and skipped.
func copyDir(src string, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return errors.New("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		return errors.New("destination already exists")
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return err
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = copyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = copyFile(srcPath, dstPath, 10000)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Optimising the copy of files in Go:
// https://opensource.com/article/18/6/copying-files-go
func copyFile(src string, dst string, BUFFERSIZE int64) error {
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

	MkdirAll(dst)

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
