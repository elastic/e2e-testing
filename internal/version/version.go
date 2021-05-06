package version

import (
	"fmt"
	"strings"
)

// Version represent an Elastic component version
type Version struct {
	original  string
	Major     string
	Minor     string
	Patch     string
	Qualifier string
}

// Ref returns the Git branch that this version refers to
func (v Version) Ref() string {
	if v.Major == "8" {
		return "master"
	} else if v.Minor != "" {
		return fmt.Sprintf("%s.%s", v.Major, v.Minor)
	} else {
		return v.original
	}
}

// String (you guessed it) returns the version as a string
func (v Version) String() string {
	return v.original
}

// Parse (you guessed it) parses the version
func Parse(version string) (*Version, error) {
	err := fmt.Errorf("%s is not a valid version", version)
	v := Version{original: version}
	parts := strings.Split(version, "-")

	switch len(parts) {
	case 2:
		v.Qualifier = parts[1]
	case 1:
	default:
		return nil, err
	}

	parts = strings.Split(parts[0], ".")
	switch len(parts) {
	case 3:
		v.Patch = parts[2]
		fallthrough
	case 2:
		v.Major = parts[0]
		v.Minor = parts[1]
	case 1:
	default:
		return nil, err
	}
	return &v, nil
}
