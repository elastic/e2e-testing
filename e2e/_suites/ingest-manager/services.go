package main

// ElasticAgentInstaller represents how to install an agent, depending of the box type
type ElasticAgentInstaller struct {
	image         string // docker image
	installCmds   []string
	name          string // the name for the binary
	path          string // the path where the agent for the binary is installed
	postInstallFn func() error
	tag           string // docker tag
}

// NewCentosInstaller returns an instance of the Centos installer
func NewCentosInstaller() ElasticAgentInstaller {
	return ElasticAgentInstaller{
		image: "centos",
		tag:   "7",
	}
}
