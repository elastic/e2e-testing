package version

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVersionParse(t *testing.T) {
	for _, testCase := range []struct {
		input   string
		version Version
		ref     string
		err     string
	}{
		{
			input: "8.0.0-SNAPSHOT",
			version: Version{
				original:  "8.0.0-SNAPSHOT",
				Major:     "8",
				Minor:     "0",
				Patch:     "0",
				Qualifier: "SNAPSHOT",
			},
			ref: "master",
		},
		{
			input: "7.x",
			version: Version{
				original: "7.x",
				Major:    "7",
				Minor:    "x",
			},
			ref: "7.x",
		},
		{
			input: "7.12.1",
			version: Version{
				original: "7.12.1",
				Major:    "7",
				Minor:    "12",
				Patch:    "1",
			},
			ref: "7.12",
		},
		{
			input: "abcdef123456",
			version: Version{
				original: "abcdef123456",
			},
			ref: "abcdef123456",
		},
		{
			input: "1.2.3.4",
			err:   "1.2.3.4 is not a valid version",
		},
		{
			input: "7.12-SNAP-SHOT",
			err:   "7.12-SNAP-SHOT is not a valid version",
		},
	} {
		version, err := Parse(testCase.input)
		if testCase.err != "" {
			assert.Equal(t, testCase.err, err.Error())
		} else {
			assert.Equal(t, &testCase.version, version)
			assert.Equal(t, nil, err)
			assert.Equal(t, testCase.ref, version.Ref())
		}
	}
}
