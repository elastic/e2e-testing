// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beats

import (
	"context"

	"github.com/elastic/e2e-testing/internal/types"
	"github.com/elastic/e2e-testing/pkg/downloads"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
)

// Operator operations that a Beat is able to perform
type Operator interface {
	Download(ctx context.Context) (string, string, error) // downloads a Beat package to the instance where the Beat runs
	Install(ctx context.Context) error                    // installs the Beat package in the instance where the Beat runs
	Logs(ctx context.Context) error                       // retrieve Beat log from the instance where the Beat runs
	Postinstall(ctx context.Context) error                // operations after the Beat is installed in the instance where the Beat runs
	Preinstall(ctx context.Context) error                 // operations before the Beat is installed in the instance where the Beat runs
	Start(ctx context.Context) error                      // start a Beat
	Stop(ctx context.Context) error                       // stop a Beat
	Uninstall(ctx context.Context) error                  // uninstall a Beat
}

// Beat struct representing an instance of a Beat
type Beat struct {
	Arch                types.Architecture        // Architecture where to install the Beat
	Docker              bool                      // whether the beat is represented by a Docker image or not (only available)
	InstallationPackage types.InstallationPackage // package to be used
	Name                string                    // name of the Beat: metricbeat, filebeat, heartbeat, etc
	OS                  types.OperativeSystem     // Operative System where to install the Beat
	Version             string                    // semantic vesion for the Beat
	XPack               bool                      // whether the Beat is an x-pack project
}

// AsDocker sets that the binary is a Docker image. It will only set the value to 'true' when the Beat has been created for
// Linux and TAR.GZ, or for Windows and ZIP
func (b *Beat) AsDocker() *Beat {
	if b.OS == types.Linux && b.InstallationPackage != types.TarGz {
		b.Docker = false
		return b
	}

	if b.OS == types.Windows && b.InstallationPackage != types.Zip {
		b.Docker = false
		return b
	}

	b.Docker = true
	return b
}

// NoXPack sets that the binary as no x-pack
func (b *Beat) NoXPack() *Beat {
	b.XPack = false
	return b
}

// GenericBeat creates an instance of an x-pack Beat
func GenericBeat(name string, os types.OperativeSystem, arch types.Architecture, osPackage types.InstallationPackage, version string) *Beat {
	if os == types.Linux {
		// Centos uses "aarch64" as architecture for ARM
		if arch == types.Arm64 && osPackage == types.Rpm {
			arch = types.Aarch64
		}
	}

	return &Beat{
		Arch:                arch,
		Docker:              false,
		Name:                name,
		OS:                  os,
		InstallationPackage: osPackage,
		Version:             version,
		XPack:               true,
	}
}

// NewLinuxBeat creates an instance of a Beat for Linux
func NewLinuxBeat(name string, osPackage types.InstallationPackage, version string) *Beat {
	return GenericBeat(name, types.Linux, types.GetArchitecture(), osPackage, version)
}

// NewMacBeat creates an instance of a Beat for Linux
func NewMacBeat(name string, version string) *Beat {
	return GenericBeat(name, types.Linux, types.GetArchitecture(), types.TarGz, version)
}

// NewWindowsBeat creates an instance of a Beat for Windows, running on AMD64 (x86_64) using a ZIP file
func NewWindowsBeat(name string, version string, xpack bool) *Beat {
	return GenericBeat(name, types.Windows, types.Amd64, types.Zip, version)
}

// Download downloads the beat
func (b *Beat) Download(ctx context.Context) (string, string, error) {
	arch := types.Architectures[b.Arch]
	artifact := b.Name
	os := types.OperativeSystems[b.OS]
	osPackage := types.InstallationPackages[b.InstallationPackage]

	span, _ := apm.StartSpanOptions(ctx, "Download "+artifact, artifact+".download", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("architecture", arch)
	span.Context.SetLabel("artifact", artifact)
	span.Context.SetLabel("os", os)
	span.Context.SetLabel("package", osPackage)
	defer span.End()

	binaryName, binaryPath, err := downloads.FetchElasticArtifact(ctx, artifact, b.Version, os, arch, osPackage, b.Docker, b.XPack)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact": artifact,
			"version":  b.Version,
			"os":       os,
			"arch":     arch,
			"package":  osPackage,
			"error":    err,
		}).Error("Could not download the binary")
		return "", "", err
	}

	return binaryName, binaryPath, nil
}

// DownloadSnapshot downloads the beat as a snapshot
func (b *Beat) DownloadSnapshot(ctx context.Context, useCISnapshots bool) (string, string, error) {
	arch := types.Architectures[b.Arch]
	artifact := b.Name
	os := types.OperativeSystems[b.OS]
	osPackage := types.InstallationPackages[b.InstallationPackage]

	span, _ := apm.StartSpanOptions(ctx, "Download "+artifact, artifact+".download", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	span.Context.SetLabel("architecture", arch)
	span.Context.SetLabel("artifact", artifact)
	span.Context.SetLabel("os", os)
	span.Context.SetLabel("package", osPackage)
	defer span.End()

	binaryName, binaryPath, err := downloads.FetchElasticArtifactForSnapshots(ctx, useCISnapshots, artifact, b.Version, os, arch, osPackage, b.Docker, b.XPack)
	if err != nil {
		log.WithFields(log.Fields{
			"artifact": artifact,
			"version":  b.Version,
			"os":       os,
			"arch":     arch,
			"package":  osPackage,
			"error":    err,
		}).Error("Could not download the snapshot binary")
		return "", "", err
	}

	return binaryName, binaryPath, nil
}

// Install installs a Beat that was previously installed in the machine
func (b *Beat) Install(ctx context.Context) error {
	log.Tracef("Executing install commands for %s", b.Name)
	return nil
}

// Logs retrieve logs for a Beat
func (b *Beat) Logs(ctx context.Context) error {
	log.Tracef("Retrieving logs for %s", b.Name)
	return nil
}

// PostInstall executes operations after a Beat is installed in the machine
func (b *Beat) PostInstall(ctx context.Context) error {
	log.Tracef("Executing additional post install commands for %s", b.Name)
	return nil
}

// PreInstall executes operations before a Beat is installed in the machine
func (b *Beat) PreInstall(ctx context.Context) error {
	log.Tracef("Executing additional pre install commands for %s", b.Name)
	return nil
}

// Start starts a Beat
func (b *Beat) Start(ctx context.Context) error {
	log.Tracef("Starting %s", b.Name)
	return nil
}

// Stop starts a Beat
func (b *Beat) Stop(ctx context.Context) error {
	log.Tracef("Stopping %s", b.Name)
	return nil
}

// Uninstall starts a Beat
func (b *Beat) Uninstall(ctx context.Context) error {
	log.Tracef("Uninstalling %s", b.Name)
	return nil
}

// ArchToString resolves the architecture to its string representation
func (b *Beat) ArchToString() string {
	return types.Architectures[b.Arch]
}

// InstallationPackageToString resolves the architecture to its string representation
func (b *Beat) InstallationPackageToString() string {
	return types.InstallationPackages[b.InstallationPackage]
}

// OSToString resolves the architecture to its string representation
func (b *Beat) OSToString() string {
	return types.OperativeSystems[b.OS]
}
