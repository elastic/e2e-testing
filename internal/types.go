// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"os"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Architecture defines an Enum for the different supported architectures
type Architecture int

const (
	// Amd64 type for AMD64, or x86_64
	Amd64 Architecture = iota
	// Aarch64 type for AARCH64
	Aarch64
	// Arm64 type for ARM64
	Arm64
)

// Architectures map with the string representation of the architectures
var Architectures = map[Architecture]string{
	Amd64:   "x86_64",
	Aarch64: "aarch64",
	Arm64:   "arm64",
}

// GetArchitecture retrieves the architecture for the underlying host, looking up the environment
// for the 'GOARCH' environment variable. If not present, it will use the runtime package for that.
// If the selected architecture is not supported in the predefined types, Amd64 will be used as fallback.
func GetArchitecture() Architecture {
	arch, present := os.LookupEnv("GOARCH")
	if !present {
		arch = runtime.GOARCH
	}

	log.Debugf("Go's architecture is (%s)", arch)

	for k, v := range Architectures {
		if strings.EqualFold(v, arch) {
			return k
		}
	}

	return Amd64
}

// OperativeSystem defines an Enum for the different supported operative systems
type OperativeSystem int

const (
	// Linux type for Linux systems
	Linux OperativeSystem = iota
	// Mac type for MacOS systems
	Mac
	// Windows type for Windows systems
	Windows
)

// OperativeSystems map with the string representation of the operative systems
var OperativeSystems = map[OperativeSystem]string{
	Linux:   "linux",
	Mac:     "darwin",
	Windows: "windows",
}

// InstallationPackage defines an Enum for the different supported installation packages
type InstallationPackage int

const (
	// Deb type for Debian packages (.deb)
	Deb InstallationPackage = iota
	// Rpm type for Centos packages (.rpm)
	Rpm
	// TarGz type for Linux packages (.tar.gz)
	TarGz
	// Zip type for Windows packages (.zip)
	Zip
)

// InstallationPackages map with the string representation of the installation packages
var InstallationPackages = map[InstallationPackage]string{
	Deb:   "deb",
	Rpm:   "rpm",
	TarGz: "tar.gz",
	Zip:   "zip",
}
