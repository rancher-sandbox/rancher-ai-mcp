package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersion(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		assert.Equal(t, Version, GetVersion())
	})

	t.Run("version without commit", func(t *testing.T) {
		Version = "v1.0.0"
		GitCommit = ""

		assert.Equal(t, "v1.0.0", GetVersion())
	})

	t.Run("version with commit", func(t *testing.T) {
		Version = "v1.0.0"
		GitCommit = "d81092997443da03f1c8b42a859bbdb998107b90"

		assert.Equal(t, "v1.0.0 (d81092997443da03f1c8b42a859bbdb998107b90)", GetVersion())
	})
}
