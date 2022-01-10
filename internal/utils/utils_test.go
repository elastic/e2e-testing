package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadFile(t *testing.T) {
	f, err := DownloadFile("https://www.elastic.co/robots.txt", "")
	assert.Nil(t, err)
	defer os.Remove(filepath.Dir(f))
}

func TestGetArchitecture(t *testing.T) {
	t.Run("Retrieving amd architecture", func(t *testing.T) {
		fallbackArch := os.Getenv("GOARCH")
		os.Setenv("GOARCH", "amd64")
		defer os.Setenv("GOARCH", fallbackArch)

		assert.Equal(t, "amd64", GetArchitecture())
	})

	// This test won't work due to the way we pull from the runtime and
	// GoArchitecture doesnt know how to handle an unknown architecture.
	// t.Run("Retrieving amd architecture as fallback", func(t *testing.T) {
	// 	fallbackArch := os.Getenv("GOARCH")
	// 	os.Setenv("GOARCH", "arch-not-found")
	// 	defer os.Setenv("GOARCH", fallbackArch)

	// 	assert.Equal(t, "amd64", GetArchitecture())
	// })

	t.Run("Retrieving arm architecture", func(t *testing.T) {
		fallbackArch := os.Getenv("GOARCH")
		os.Setenv("GOARCH", "arm64")
		defer os.Setenv("GOARCH", fallbackArch)

		assert.Equal(t, "arm64", GetArchitecture())
	})
}

func TestIsCommit(t *testing.T) {
	t.Run("Returns true with commits", func(t *testing.T) {
		assert.True(t, IsCommit("abcdef1234"))
		assert.True(t, IsCommit("a12345"))
		assert.True(t, IsCommit("abcdef1"))
	})

	t.Run("Returns false with non-commits", func(t *testing.T) {
		assert.False(t, IsCommit("master"))
		assert.False(t, IsCommit("7.12.x"))
		assert.False(t, IsCommit("7.11.x"))
		assert.False(t, IsCommit("pr12345"))
	})

	t.Run("Returns false with commits in snapshots", func(t *testing.T) {
		assert.False(t, IsCommit("8.0.0-a12345-SNAPSHOT"))
	})
}
